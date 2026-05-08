package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

func TestAnnouncementStoreCreateAndUpdatePersistModeratorUUIDs(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	t.Cleanup(func() { testDB.Close() })
	t.Cleanup(func() { testDB.Cleanup(context.Background()) })

	db := &DB{DB: testDB.DB}
	userStore := NewUserStore(db)
	announcementStore := NewAnnouncementStore(db)

	ctx := context.Background()

	actor, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "announcements-admin@example.com",
		DisplayName: "Announcements Admin",
	})
	if err != nil {
		t.Fatalf("create actor user: %v", err)
	}

	created, err := announcementStore.Create(ctx, actor.ID, models.SaveAnnouncementParams{
		Title:       "Launch",
		Body:        "Body copy",
		Status:      models.AnnouncementStatusPublished,
		Priority:    10,
		Placements:  []models.AnnouncementPlacement{models.AnnouncementPlacementHome},
		Audience:    models.AnnouncementAudienceAll,
		Dismissible: true,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}

	var createdByID, updatedByID sql.NullString
	if err := testDB.QueryRowContext(ctx,
		`SELECT created_by_user_id::text, updated_by_user_id::text FROM announcements WHERE id = $1`,
		created.ID,
	).Scan(&createdByID, &updatedByID); err != nil {
		t.Fatalf("query created announcement audit fields: %v", err)
	}

	if !createdByID.Valid || createdByID.String != actor.ID {
		t.Fatalf("created_by_user_id = %+v, want %q", createdByID, actor.ID)
	}
	if !updatedByID.Valid || updatedByID.String != actor.ID {
		t.Fatalf("updated_by_user_id = %+v, want %q", updatedByID, actor.ID)
	}

	updated, err := announcementStore.Update(ctx, created.ID, actor.ID, models.SaveAnnouncementParams{
		Title:       "Launch updated",
		Body:        "Updated body copy",
		Status:      models.AnnouncementStatusPublished,
		Priority:    25,
		Placements:  []models.AnnouncementPlacement{models.AnnouncementPlacementNews},
		Audience:    models.AnnouncementAudienceSignedIn,
		Dismissible: false,
	})
	if err != nil {
		t.Fatalf("update announcement: %v", err)
	}
	if updated == nil {
		t.Fatal("update announcement returned nil item")
	}

	if err := testDB.QueryRowContext(ctx,
		`SELECT updated_by_user_id::text FROM announcements WHERE id = $1`,
		created.ID,
	).Scan(&updatedByID); err != nil {
		t.Fatalf("query updated announcement audit field: %v", err)
	}

	if !updatedByID.Valid || updatedByID.String != actor.ID {
		t.Fatalf("updated_by_user_id after update = %+v, want %q", updatedByID, actor.ID)
	}
}
