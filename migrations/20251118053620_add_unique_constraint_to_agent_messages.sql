-- Clean up duplicate entries before adding unique constraint
-- 保留每組 (agent_campaign_id, agent_id) 的最早記錄，刪除其他重複記錄
-- DELETE am1 FROM `agent_messages` am1
-- JOIN (
--     SELECT agent_campaign_id, agent_id, MIN(id) as min_id
--     FROM `agent_messages`
--     WHERE deleted_at IS NULL
--     GROUP BY agent_campaign_id, agent_id
--     HAVING COUNT(*) > 1
-- ) am2 ON am1.agent_campaign_id = am2.agent_campaign_id
--     AND am1.agent_id = am2.agent_id
--     AND am1.id > am2.min_id
-- WHERE am1.deleted_at IS NULL;

-- Modify "agent_messages" table
ALTER TABLE `agent_messages` ADD UNIQUE INDEX `unique_agent_campaign_message` (`agent_campaign_id`, `agent_id`);
