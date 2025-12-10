-- Modify "agent_relationships" table
ALTER TABLE `agent_relationships` ADD INDEX `idx_agent_relationships_child_id` (`child_id`), ADD INDEX `idx_agent_relationships_parent_id` (`parent_id`);
