package avs

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	parentEvaluationID = 42
	evaluationName     = "test_evaluation"
)

func TestClient_CreateEvaluation(t *testing.T) {
	t.Run("create evaluation the first time", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
		}, logrus.New())
		assert.NoError(t, err)

		// When
		response, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     evaluationName,
			ParentId: parentEvaluationID,
		})

		// Then
		assert.NoError(t, err)
		assert.Equal(t, evaluationName, response.Name)
		assert.NotEmpty(t, server.Evaluations.ParentIDrefs[parentEvaluationID])
	})

	t.Run("create evaluation with token reset", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		server.TokenExpired = 1
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
		}, logrus.New())
		assert.NoError(t, err)

		// When
		response, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     evaluationName,
			ParentId: parentEvaluationID,
		})

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "test_evaluation", response.Name)
		assert.NotEmpty(t, server.Evaluations.ParentIDrefs[parentEvaluationID])
	})

	t.Run("401 error during creating evaluation", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		server.TokenExpired = 2
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		_, err = client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name: "test_evaluation",
		})

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

func TestClient_DeleteEvaluation(t *testing.T) {
	t.Run("should delete existing evaluation", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name: "test_evaluation",
		})
		assert.NoError(t, err)

		// When
		err = client.DeleteEvaluation(resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.Evaluations.BasicEvals)
	})

	t.Run("should return error when trying to delete not existing evaluation", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		_, err = client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})
		assert.NoError(t, err)

		// When
		err = client.DeleteEvaluation(123)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.Evaluations.BasicEvals[123])
	})
}

func TestClient_GetEvaluation(t *testing.T) {
	t.Run("should get existing evaluation", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:        "test_evaluation_create",
			Description: "custom description",
		})
		assert.NoError(t, err)

		// When
		getResp, err := client.GetEvaluation(resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, getResp, resp)
	})

	t.Run("should return error when trying to get not existing evaluation", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		_, err = client.GetEvaluation(1)

		// Then
		assert.Contains(t, err.Error(), "404")
	})
}

func TestClient_Status(t *testing.T) {
	t.Run("should get status", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{})
		assert.NoError(t, err)

		// When
		getResp, err := client.GetEvaluation(resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, getResp.Status, StatusActive)
	})

	t.Run("should update status", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{})
		assert.NoError(t, err)

		// When
		resp, err = client.SetStatus(resp.Id, StatusDeleted)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, resp.Status, StatusDeleted)
	})

	t.Run("should not update invalid status", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{})
		assert.NoError(t, err)

		// When
		resp, err = client.SetStatus(resp.Id, "")

		// Then
		assert.Contains(t, err.Error(), "500")
	})
}

func TestClient_RemoveReferenceFromParentEval(t *testing.T) {
	t.Run("should remove reference from parent eval", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})
		assert.NoError(t, err)

		// When
		err = client.RemoveReferenceFromParentEval(parentEvaluationID, resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.Evaluations.ParentIDrefs[parentEvaluationID])
	})
	t.Run("should return error when wrong api url provided", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", "http://not-existing"),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		err = client.RemoveReferenceFromParentEval(parentEvaluationID, 111)

		// then
		assert.Error(t, err)
	})
	t.Run("should return error when parent evaluation does not contain subevaluation", func(t *testing.T) {
		// Given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		err = client.RemoveReferenceFromParentEval(int64(9999), 111)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

}

func TestClient_AddTag(t *testing.T) {
	t.Run("should add tag to existing evaluation", func(t *testing.T) {
		// given
		server := NewMockAvsServer(t)
		mockServer := FixMockAvsServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		response, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})
		assert.NoError(t, err)

		fixedTag := FixTag()

		// when
		eval, err := client.AddTag(response.Id, fixedTag)

		// then
		assert.NoError(t, err)
		assert.Equal(t, fixedTag, eval.Tags[0])
	})
}
