
ALTER TABLE journals CHANGE prompt notification text;
ALTER TABLE prompts CHANGE prompt_id notification_id varchar(36);
ALTER TABLE user_prompts CHANGE prompt_id notification_id varchar(36);
ALTER TABLE user_prompts CHANGE next_prompt_time next_notification_time datetime(6);
RENAME TABLE prompts TO notifications;
RENAME TABLE user_prompts TO user_notifications;

