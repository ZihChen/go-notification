-- Modify "agent_campaigns" table
ALTER TABLE `agent_campaigns` ADD COLUMN `schedule_type` enum('scheduled','immediate') NOT NULL DEFAULT "scheduled" AFTER `scheduled_at`;
