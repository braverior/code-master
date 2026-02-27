-- CodeMaster Database Schema
-- MySQL 8.0+, charset=utf8mb4

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `feishu_uid` VARCHAR(128) NOT NULL,
  `feishu_union_id` VARCHAR(128) DEFAULT NULL,
  `name` VARCHAR(64) NOT NULL,
  `avatar` VARCHAR(512) DEFAULT NULL,
  `email` VARCHAR(128) DEFAULT NULL,
  `role` VARCHAR(10) NOT NULL DEFAULT 'rd',
  `status` TINYINT NOT NULL DEFAULT 1,
  `last_login_at` TIMESTAMP NULL DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_feishu_uid` (`feishu_uid`),
  UNIQUE KEY `idx_feishu_union_id` (`feishu_union_id`),
  KEY `idx_role` (`role`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `projects` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `description` TEXT,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `doc_links` JSON DEFAULT NULL,
  `status` VARCHAR(10) NOT NULL DEFAULT 'active',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_owner_id` (`owner_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `project_members` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role` VARCHAR(10) NOT NULL,
  `joined_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_user` (`project_id`, `user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `repositories` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `git_url` VARCHAR(512) NOT NULL,
  `platform` VARCHAR(10) NOT NULL,
  `platform_project_id` VARCHAR(64) DEFAULT NULL,
  `default_branch` VARCHAR(64) NOT NULL DEFAULT 'develop',
  `access_token` VARCHAR(512) DEFAULT NULL,
  `analysis_result` JSON DEFAULT NULL,
  `analysis_status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `analyzed_at` TIMESTAMP NULL DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_project_id` (`project_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `requirements` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT UNSIGNED NOT NULL,
  `title` VARCHAR(256) NOT NULL,
  `description` TEXT NOT NULL,
  `doc_links` JSON DEFAULT NULL,
  `doc_content` LONGTEXT DEFAULT NULL,
  `priority` VARCHAR(5) NOT NULL DEFAULT 'p1',
  `status` VARCHAR(20) NOT NULL DEFAULT 'draft',
  `creator_id` BIGINT UNSIGNED NOT NULL,
  `assignee_id` BIGINT UNSIGNED DEFAULT NULL,
  `repository_id` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_project_id` (`project_id`),
  KEY `idx_status` (`status`),
  KEY `idx_creator_id` (`creator_id`),
  KEY `idx_assignee_id` (`assignee_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `codegen_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `requirement_id` BIGINT UNSIGNED NOT NULL,
  `repository_id` BIGINT UNSIGNED NOT NULL,
  `source_branch` VARCHAR(64) NOT NULL,
  `target_branch` VARCHAR(128) NOT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `prompt` TEXT DEFAULT NULL,
  `output_log` LONGTEXT DEFAULT NULL,
  `diff_stat` JSON DEFAULT NULL,
  `error_message` TEXT DEFAULT NULL,
  `claude_cost_usd` DECIMAL(10,4) DEFAULT NULL,
  `started_at` TIMESTAMP NULL DEFAULT NULL,
  `completed_at` TIMESTAMP NULL DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_requirement_id` (`requirement_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `code_reviews` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `codegen_task_id` BIGINT UNSIGNED NOT NULL,
  `ai_review_result` JSON DEFAULT NULL,
  `ai_score` INT DEFAULT NULL,
  `ai_status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `human_reviewer_id` BIGINT UNSIGNED DEFAULT NULL,
  `human_comment` TEXT DEFAULT NULL,
  `human_status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `merge_request_id` VARCHAR(64) DEFAULT NULL,
  `merge_request_url` VARCHAR(512) DEFAULT NULL,
  `merge_status` VARCHAR(10) NOT NULL DEFAULT 'none',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_codegen_task_id` (`codegen_task_id`),
  KEY `idx_human_reviewer_id` (`human_reviewer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED DEFAULT NULL,
  `action` VARCHAR(64) NOT NULL,
  `resource_type` VARCHAR(32) NOT NULL,
  `resource_id` BIGINT UNSIGNED DEFAULT NULL,
  `detail` JSON DEFAULT NULL,
  `ip` VARCHAR(45) DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_resource` (`resource_type`, `resource_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
