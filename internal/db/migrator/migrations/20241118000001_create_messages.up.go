package migrations

import (
	"context"

	"github.com/boratanrikulu/sendpulse/internal/db"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, bunDB *bun.DB) error {
		if _, err := bunDB.NewCreateTable().Model((*db.Message)(nil)).Exec(ctx); err != nil {
			return err
		}

		// Create indexes for better performance
		if _, err := bunDB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status)"); err != nil {
			return err
		}

		if _, err := bunDB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)"); err != nil {
			return err
		}

		if _, err := bunDB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at)"); err != nil {
			return err
		}

		// Add check constraint for content length
		if _, err := bunDB.Exec("ALTER TABLE messages ADD CONSTRAINT check_content_length CHECK (length(content) <= 1000)"); err != nil {
			return err
		}

		// Add check constraint for valid phone number format (basic validation)
		if _, err := bunDB.Exec(`ALTER TABLE messages ADD CONSTRAINT check_phone_format CHECK ("to" ~ '^\+[1-9]\d{1,14}$')`); err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, bunDB *bun.DB) error {
		if _, err := bunDB.NewDropTable().Model((*db.Message)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}
