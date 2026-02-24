package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMutationCRUD_UpdateAndDeleteByUUID_AcrossEntities(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	userID := uuid.New().String()
	_, err := eng.Insert("User").
		Set("id", userID).
		Set("email", "crud-user@mail.com").
		Set("name", "CRUD User").
		Execute(ctx)
	require.NoError(t, err)

	orderID := uuid.New().String()
	_, err = eng.Insert("Order").
		Set("id", orderID).
		Set("total", 99.5).
		Set("status", "pending").
		Set("user_id", userID).
		Execute(ctx)
	require.NoError(t, err)

	updatedUser, err := eng.Update("User").
		Filter("id", "eq", userID).
		Set("name", "CRUD User Updated").
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, updatedUser.Affected)
	require.Len(t, updatedUser.Records, 1)
	require.Equal(t, "CRUD User Updated", updatedUser.Records[0]["name"])

	updatedOrder, err := eng.Update("Order").
		Filter("id", "eq", orderID).
		Set("status", "completed").
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, updatedOrder.Affected)
	require.Len(t, updatedOrder.Records, 1)
	require.Equal(t, "completed", updatedOrder.Records[0]["status"])

	deletedOrder, err := eng.Delete("Order").
		Filter("id", "eq", orderID).
		Filter("user_id", "eq", userID).
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deletedOrder.Affected)

	deletedUser, err := eng.Delete("User").
		Filter("id", "eq", userID).
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deletedUser.Affected)
}
