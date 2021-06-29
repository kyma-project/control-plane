package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type FakeTokenSource string

var fixToken FakeTokenSource = "fake-token-1234"

func (t FakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: string(t),
		Expiry:      time.Now().Add(time.Duration(12 * time.Hour)),
	}, nil
}

var runtime1 = fixRuntimeDTO("runtime1")
var runtime2 = fixRuntimeDTO("runtime2")
var runtime3 = fixRuntimeDTO("runtime3")

func TestClient_ListRuntimes(t *testing.T) {
	t.Run("test request URL and response are correct", func(t *testing.T) {
		//given
		called := 0
		params := ListParameters{
			Page:             1,
			PageSize:         50,
			OperationDetail:  LastOperation,
			KymaConfig:       true,
			ClusterConfig:    true,
			GlobalAccountIDs: []string{"sa1", "ga2"},
			SubAccountIDs:    []string{"sa1", "sa2"},
			InstanceIDs:      []string{"id1", "id2"},
			RuntimeIDs:       []string{"rid1", "rid2"},
			Regions:          []string{"region1", "region2"},
			Shoots:           []string{"shoot1", "shoot2"},
			Plans:            []string{"plan1", "plan2"},
			States:           []State{StateFailed, StateSucceeded},
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/runtimes", r.URL.Path)
			assert.Equal(t, r.Header.Get("Authorization"), fmt.Sprintf("Bearer %s", fixToken))
			query := r.URL.Query()
			assert.ElementsMatch(t, []string{strconv.Itoa(params.Page)}, query[pagination.PageParam])
			assert.ElementsMatch(t, []string{strconv.Itoa(params.PageSize)}, query[pagination.PageSizeParam])
			assert.ElementsMatch(t, []string{string(LastOperation)}, query[OperationDetailParam])
			assert.ElementsMatch(t, []string{"true"}, query[KymaConfigParam])
			assert.ElementsMatch(t, []string{"true"}, query[ClusterConfigParam])
			assert.ElementsMatch(t, params.GlobalAccountIDs, query[GlobalAccountIDParam])
			assert.ElementsMatch(t, params.SubAccountIDs, query[SubAccountIDParam])
			assert.ElementsMatch(t, params.InstanceIDs, query[InstanceIDParam])
			assert.ElementsMatch(t, params.RuntimeIDs, query[RuntimeIDParam])
			assert.ElementsMatch(t, params.Regions, query[RegionParam])
			assert.ElementsMatch(t, params.Shoots, query[ShootParam])
			assert.ElementsMatch(t, params.Plans, query[PlanParam])
			stateParams := query[StateParam]
			assert.Len(t, stateParams, 2)
			assert.EqualValues(t, params.States[0], stateParams[0])
			assert.EqualValues(t, params.States[1], stateParams[1])

			err := respondRuntimes(w, []RuntimeDTO{runtime1, runtime2}, 2)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		//when
		rp, err := client.ListRuntimes(params)

		//then
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, 2, rp.Count)
		assert.Equal(t, 2, rp.TotalCount)
		assert.Len(t, rp.Data, 2)
		assert.Equal(t, rp.Data[0].InstanceID, runtime1.InstanceID)
		assert.Equal(t, rp.Data[1].InstanceID, runtime2.InstanceID)
	})

	t.Run("test pagination", func(t *testing.T) {
		called := 0
		params := ListParameters{
			PageSize: 2,
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++

			err := respondRuntimes(w, []RuntimeDTO{runtime1, runtime2}, 4)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		//when
		rp, err := client.ListRuntimes(params)

		//then
		require.NoError(t, err)
		assert.Equal(t, 2, called)
		assert.Equal(t, 4, rp.Count)
		assert.Equal(t, 4, rp.TotalCount)
		assert.Len(t, rp.Data, 4)
	})
}

func fixRuntimeDTO(id string) RuntimeDTO {
	return RuntimeDTO{
		InstanceID:       id,
		RuntimeID:        id,
		GlobalAccountID:  id,
		SubAccountID:     id,
		ProviderRegion:   id,
		SubAccountRegion: id,
		ShootName:        id,
		ServiceClassID:   id,
		ServiceClassName: id,
		ServicePlanID:    id,
		ServicePlanName:  id,
		Status: RuntimeStatus{
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
			Provisioning: &Operation{
				State:       id,
				Description: id,
			},
		},
	}
}

func respondRuntimes(w http.ResponseWriter, runtimes []RuntimeDTO, totalCount int) error {
	rp := RuntimesPage{
		Data:       runtimes,
		Count:      len(runtimes),
		TotalCount: totalCount,
	}
	data, err := json.Marshal(rp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}
