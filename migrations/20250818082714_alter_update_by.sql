-- Modify "message_campaign" table
ALTER TABLE `message_campaign` DROP COLUMN `size:255`, ADD COLUMN `updated_by` varchar(255) NULL;
