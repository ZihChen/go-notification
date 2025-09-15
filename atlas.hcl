# ================================================================
# Atlas Configuration for Fat Notification Cat
# 支援兩種環境配置: local (本地開發), deployment (Kubernetes部署)
# ================================================================

# 通用 Schema 數據源
data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/infrastructure/models",
    "--dialect", "mysql",
  ]
}

# ================================================================
# Local Development Environment (使用 .env 文件)
# ================================================================
env "local" {
  src = data.external_schema.gorm.url
  
  # 從環境變數構建連接字串
  url = format("mysql://%s:%s@%s:%s/%s?tls=true",
    var.db_user,
    var.db_password,
    var.db_host,
    var.db_port,
    var.db_name
  )
  
  # 使用變數構建測試資料庫連接字串
  dev = format("mysql://%s:%s@%s:%s/%s?tls=true",
    var.atlas_dev_user,
    var.atlas_dev_password,
    var.db_host,
    var.db_port,
    "ms_fatidentitycat"  # dev 資料庫名稱固定
  )

  migration {
    dir = "file://migrations"
    lock_timeout = "10s"
  }
  
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
  
  diff {
    skip {
      drop_schema = true
      drop_table = true
    }
  }
  
  exclude = [
    "atlas_schema_revisions"
  ]
}

# ================================================================
# Deployment Environment (使用 Kubernetes ConfigMap/Secrets)
# ================================================================
env "deployment" {
  src = data.external_schema.gorm.url
  
  # 從環境變數構建連接字串 (由 ConfigMap/Secrets 提供)
  url = format("mysql://%s:%s@%s:%s/%s?tls=true",
    var.db_user,
    var.db_password,
    var.db_host,
    var.db_port,
    var.db_name
  )

  migration {
    dir = "file://migrations"
    lock_timeout = "60s"
    baseline = "1" # 部署環境建議設置基準版本
  }
  
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
  
  exclude = [
    "atlas_schema_revisions"
  ]
}

# ================================================================
# 向後兼容：保留原 gorm 環境 (映射到 local)
# ================================================================
env "gorm" {
  src = data.external_schema.gorm.url
  
  url = format("mysql://%s:%s@%s:%s/%s?tls=true",
    var.db_user,
    var.db_password,
    var.db_host,
    var.db_port,
    var.db_name
  )
  
  # 使用變數構建測試資料庫連接字串
  dev = format("mysql://%s:%s@%s:%s/%s?tls=true",
    var.atlas_dev_user,
    var.atlas_dev_password,
    var.db_host,
    var.db_port,
    "ms_fatidentitycat"  # dev 資料庫名稱固定
  )

  migration {
    dir = "file://migrations"
    lock_timeout = "5s"
  }
  
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
  
  exclude = [
    "atlas_schema_revisions"
  ]
}

# ================================================================
# 環境變數定義
# ================================================================
variable "db_host" {
  type = string
  default = "localhost"
}

variable "db_port" {
  type = string
  default = "3306"
}

variable "db_user" {
  type = string
  default = "root"
}

variable "db_password" {
  type = string
  default = "password"
}

variable "db_name" {
  type = string
  default = "fat_notification_cat"
}

# Atlas Dev 資料庫變數
variable "atlas_dev_user" {
  type = string
  default = "dev_user"
}

variable "atlas_dev_password" {
  type = string
  default = "dev_password"
}
