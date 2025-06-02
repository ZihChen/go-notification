-- 初始化資料庫結構

-- 商戶表
CREATE TABLE merchants (
    id BIGINT NOT NULL AUTO_INCREMENT,
    global_merchant_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_global_merchant_id (global_merchant_id),
    UNIQUE KEY uk_name (name),
    UNIQUE KEY uk_api_key (api_key)
);

-- 玩家表
CREATE TABLE players (
    id BIGINT NOT NULL AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL,
    global_player_id VARCHAR(100) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    account VARCHAR(255) NOT NULL,
    email VARCHAR(255) NULL,
    last_active_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_global_player_id (global_player_id),
    UNIQUE KEY uk_api_key (api_key)
);

-- 管理員表
CREATE TABLE managers (
    id BIGINT NOT NULL AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL,
    global_manager_id VARCHAR(100) NOT NULL,
    account VARCHAR(255) NOT NULL,
    email VARCHAR(255) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_global_manager_id (global_manager_id)
);

-- 訊息活動表
CREATE TABLE message_campaigns (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    category TINYINT NOT NULL COMMENT '1=member, 2=bonus, 3=others',
    item TINYINT NOT NULL COMMENT '1=registration, 2=identity_verification, ..., 6=all',
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL COMMENT '可包含 HTML Tag',
    focus TINYINT NOT NULL COMMENT '1=_in_thirty, 2=_low_activity, 3=_not_activity, 8=_one, 9=_level, 10=_tag, 11=_all',
    auto_send BOOLEAN DEFAULT FALSE COMMENT '是否為系統自動訊息',
    total_target_count INT DEFAULT 0 COMMENT '預估發送對象總人數',
    real_sent_count INT DEFAULT 0 COMMENT '實際成功發送人數',
    send_start_time DATETIME DEFAULT NULL,
    send_end_time DATETIME DEFAULT NULL,
    created_by VARCHAR(100) NOT NULL COMMENT '建立者帳號或名稱',
    updated_by VARCHAR(100) DEFAULT NULL COMMENT '最後更新者帳號或名稱',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);


-- 玩家訊息表
CREATE TABLE player_messages (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    global_player_id VARCHAR(100) NOT NULL,
    campaign_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_global_player_id (global_player_id)
);


