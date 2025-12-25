package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/fluxionwatt/gridbeat/core"
	"github.com/fluxionwatt/gridbeat/internal/db"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// resetPasswordCmd resets root password to "admin" and exits.
// resetPasswordCmd 将 root 密码重置为 "admin" 然后退出。
func resetPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-password",
		Short: "Reset root password to admin",
		RunE: func(cmd *cobra.Command, args []string) error {
			//cfg, err := config.Load(cfgFile)
			//if err != nil {
			//	return err
			//}

			logger := logrus.New()

			cfg := &core.Gconfig

			gdb, err := db.Open(cfg, logger)
			if err != nil {
				return err
			}
			if err := models.Migrate(gdb); err != nil {
				return err
			}

			// Ensure root exists ("admin" password by default if created).
			// 确保 root 存在（首次创建默认密码 admin）。
			if err := models.EnsureRootUser(gdb); err != nil {
				return err
			}

			var root models.User
			if err := gdb.Where("username = ?", "root").First(&root).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("root user missing")
				}
				return err
			}

			hash, err := util.HashPassword("admin")
			if err != nil {
				return err
			}

			root.PasswordHash = hash
			root.UpdatedAt = time.Now()
			if err := gdb.Save(&root).Error; err != nil {
				return err
			}

			fmt.Println("root password reset to: admin.")
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(resetPasswordCmd())
}
