-- Create "agent_campaigns" table
CREATE TABLE `agent_campaigns` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `title` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `scheduled_at` datetime NOT NULL,
  `status` enum('draft','scheduled','sent','failed','cancelled') NOT NULL DEFAULT "draft",
  `target_type` enum('all','specific','line') NOT NULL,
  `target_details` text NULL,
  `target_count` bigint NULL DEFAULT 0,
  `real_sent_count` bigint NULL DEFAULT 0,
  `created_by` varchar(255) NOT NULL,
  `updated_by` varchar(255) NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_agent_campaigns_created_by` (`created_by`),
  INDEX `idx_agent_campaigns_deleted_at` (`deleted_at`),
  INDEX `idx_agent_campaigns_merchant_id` (`merchant_id`),
  INDEX `idx_agent_campaigns_scheduled_at` (`scheduled_at`),
  INDEX `idx_agent_campaigns_status` (`status`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "agent_messages" table
CREATE TABLE `agent_messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `agent_campaign_id` bigint unsigned NOT NULL,
  `agent_id` bigint unsigned NOT NULL,
  `is_read` bool NULL DEFAULT 0,
  `read_at` datetime NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_agent_messages_agent_campaign_id` (`agent_campaign_id`),
  INDEX `idx_agent_messages_agent_id` (`agent_id`),
  INDEX `idx_agent_messages_created_at` (`created_at`),
  INDEX `idx_agent_messages_deleted_at` (`deleted_at`),
  INDEX `idx_agent_messages_is_read` (`is_read`),
  INDEX `idx_agent_messages_read_at` (`read_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "agent_relationships" table
CREATE TABLE `agent_relationships` (
  `parent_id` bigint unsigned NOT NULL,
  `child_id` bigint unsigned NOT NULL,
  `depth_level` bigint NOT NULL,
  `path_hash` varchar(64) NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`parent_id`, `child_id`),
  INDEX `idx_agent_relationships_depth_level` (`depth_level`),
  INDEX `idx_agent_relationships_path_hash` (`path_hash`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "agents" table
CREATE TABLE `agents` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `global_agent_id` varchar(255) NOT NULL,
  `account` varchar(255) NOT NULL,
  `ancestry` text NULL,
  `current_sign_in_at` datetime NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_agents_account` (`account`),
  INDEX `idx_agents_current_sign_in_at` (`current_sign_in_at`),
  INDEX `idx_agents_deleted_at` (`deleted_at`),
  INDEX `idx_agents_merchant_id` (`merchant_id`),
  INDEX `idx_agents_updated_at` (`updated_at`),
  UNIQUE INDEX `unique_global_agent_id` (`global_agent_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
