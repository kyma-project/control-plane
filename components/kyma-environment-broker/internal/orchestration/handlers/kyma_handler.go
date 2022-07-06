package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"golang.org/x/mod/semver"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/google/go-github/github"
)

type kymaHandler struct {
	orchestrations storage.Orchestrations
	queue          *process.Queue
	converter      Converter
	gitClient      *github.Client
	log            logrus.FieldLogger
}

func NewKymaHandler(orchestrations storage.Orchestrations, q *process.Queue, log logrus.FieldLogger) *kymaHandler {
	return &kymaHandler{
		orchestrations: orchestrations,
		queue:          q,
		log:            log,
		converter:      Converter{},
		gitClient:      github.NewClient(nil),
	}
}

func (h *kymaHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/upgrade/kyma", h.createOrchestration).Methods(http.MethodPost)
}

func (h *kymaHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	// validate request body
	params := orchestration.Parameters{}
	if r.Body != nil {
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			h.log.Errorf("while decoding request body: %v", err)
			httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while decoding request body"))
			return
		}
	}

	// validate target
	err := validateTarget(params.Targets)
	if err != nil {
		h.log.Errorf("while validating target: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while validating target"))
		return
	}

	// validate Kyma version
	err = h.ValidateKymaVersion(params.Kyma.Version)
	if err != nil {
		h.log.Errorf("while validating kyma version: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while validating kyma version"))
		return
	}

	// validate deprecated parameteter `maintenanceWindow`
	err = h.ValidateDeprecatedParameters(params)
	if err != nil {
		h.log.Errorf("found deprecated value: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "found deprecated value"))
		return
	}

	// validate `schedule` field
	err = h.ValidateScheduleParameter(&params)
	if err != nil {
		h.log.Errorf("found deprecated value: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "found deprecated value"))
		return
	}

	// defaults strategy if not specified to Parallel with Immediate schedule
	defaultOrchestrationStrategy(&params.Strategy)

	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		Type:            orchestration.UpgradeKymaOrchestration,
		State:           orchestration.Pending,
		Description:     "queued for processing",
		Parameters:      params,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = h.orchestrations.Insert(o)
	if err != nil {
		h.log.Errorf("while inserting orchestration to storage: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while inserting orchestration to storage"))
		return
	}

	h.queue.Add(o.OrchestrationID)

	response := orchestration.UpgradeResponse{OrchestrationID: o.OrchestrationID}

	httputil.WriteResponse(w, http.StatusAccepted, response)
}

// ValidateKymaVersion validates provided version. Supports three types of versioning:
// semantic version, PR-<number>, and <branch name>-<commit hash>.
// Validates version iff GitHub responded with 4xx code. If GitHub API does not work
// (e.g. due to API RATE limit), returns version as valid.
func (h *kymaHandler) ValidateKymaVersion(version string) error {
	var (
		err          error
		resp         *github.Response
		shouldHandle = func(resp *github.Response) bool {
			return resp != nil &&
				resp.StatusCode >= 400 && resp.StatusCode < 500 &&
				resp.StatusCode != http.StatusForbidden
		}
	)

	switch {
	// handle semantic version
	case semver.IsValid(fmt.Sprintf("v%s", version)):
		_, resp, err = h.gitClient.Repositories.GetReleaseByTag(context.Background(), internal.GitKymaProject, internal.GitKymaRepo, version)
	// handle PR-<number>
	case strings.HasPrefix(version, "PR-"):
		prID, _ := strconv.Atoi(strings.TrimPrefix(version, "PR-"))
		_, resp, err = h.gitClient.PullRequests.Get(context.Background(), internal.GitKymaProject, internal.GitKymaRepo, prID)
	// handle <branch name>-<commit hash>
	case strings.Contains(version, "-"):
		chunks := strings.Split(version, "-")
		branch, commit := strings.Join(chunks[:len(chunks)-1], "-"), chunks[len(chunks)-1]

		// get diff from the branch head to commit
		var diff *github.CommitsComparison
		diff, resp, err = h.gitClient.Repositories.CompareCommits(context.Background(), internal.GitKymaProject, internal.GitKymaRepo, branch, commit)

		// if diff contains commits, the searched commit is not on the given branch
		if diff != nil && len(diff.Commits) > 0 || shouldHandle(resp) {
			return fmt.Errorf("invalid Kyma version, commit %s not present on branch %s", commit, branch)
		}
	}

	// handle iff GitHub API responded
	if shouldHandle(resp) {
		return errors.Wrapf(err, "invalid Kyma version, version %s not found", version)
	}

	return nil
}

// ValidateDeprecatedParameters cheks if `maintenanceWindow` parameter is used as schedule.
func (h *kymaHandler) ValidateDeprecatedParameters(params orchestration.Parameters) error {
	if params.Strategy.Schedule == string(orchestration.MaintenanceWindow) {
		return fmt.Errorf("{\"strategy\":{\"schedule\": \"maintenanceWindow\"} is deprecated use {\"strategy\":{\"MaintenanceWindow\": true} instead")
	}
	return nil
}

// ValidateScheduleParameter cheks if the schedule parameter is valid.
func (h *kymaHandler) ValidateScheduleParameter(params *orchestration.Parameters) error {
	switch params.Strategy.Schedule {
	case "immediate":
	case "now":
		params.Strategy.ScheduleTime = time.Now()
	default:
		parsedTime, err := time.Parse(time.RFC3339, params.Strategy.Schedule)
		if err == nil {
			params.Strategy.ScheduleTime = parsedTime
		} else {
			return fmt.Errorf("the schedule filed does not contain `imediate`/`now` nor is a date: %w", err)
		}
	}
	return nil
}
