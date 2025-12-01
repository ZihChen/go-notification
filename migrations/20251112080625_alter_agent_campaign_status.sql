-- Modify "agent_campaigns" table
ALTER TABLE `agent_campaigns` MODIFY COLUMN `status` enum('draft','scheduled','sending','sent','failed','cancelled') NOT NULL DEFAULT "draft";
