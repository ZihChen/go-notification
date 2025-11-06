package main

import (
	_ "ariga.io/atlas-provider-gorm/gormschema"
	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	_ "github.com/jvdiamondtech/ms-notification-cat/cmd/consumer"
	_ "github.com/jvdiamondtech/ms-notification-cat/cmd/migrate"
	_ "github.com/jvdiamondtech/ms-notification-cat/cmd/scheduler"
	_ "github.com/jvdiamondtech/ms-notification-cat/cmd/web"
	_ "github.com/jvdiamondtech/ms-notification-cat/cmd/worker"
	_ "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
)

// @title Notification Service API
// @version 1.0
// @description 用於管理商戶、玩家和管理員的通知服務
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.jvdiamondtech.com/support
// @contact.email support@jvdiamondtech.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name API-Key
func main() {
	cmd.Execute()
}
