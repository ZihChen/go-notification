-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `trigger_type` varchar(20) NOT NULL DEFAULT "success" AFTER `item`;
