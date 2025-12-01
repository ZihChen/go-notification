-- Modify "message_campaign" table
ALTER TABLE `message_campaign`
    ADD COLUMN `app_content` varchar(255) NULL AFTER `content`,
    ADD COLUMN `notification_types` tinyint unsigned NOT NULL DEFAULT 1 AFTER `app_content`;
-- Create "push_keys" table
CREATE TABLE `push_keys` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `global_merchant_id` varchar(100) NOT NULL,
  `merchant_id` bigint unsigned NOT NULL,
  `key` varchar(100) NOT NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_push_keys_global_merchant_id` (`global_merchant_id`),
  INDEX `idx_push_keys_merchant_id` (`merchant_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
