-- OpenManus-Go Database Initialization Script
-- 创建 OpenManus-Go 所需的数据库表和初始数据

-- 设置字符集和校对规则
SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- 使用 openmanus 数据库
USE openmanus;

-- 1. Agent 执行轨迹表
CREATE TABLE IF NOT EXISTS agent_traces (
    id VARCHAR(255) PRIMARY KEY,
    goal TEXT NOT NULL,
    status ENUM('running', 'completed', 'failed', 'canceled') NOT NULL DEFAULT 'running',
    start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP NULL,
    duration_ms BIGINT DEFAULT 0,
    step_count INT DEFAULT 0,
    token_count INT DEFAULT 0,
    final_result TEXT,
    error_message TEXT,
    agent_type VARCHAR(100) DEFAULT 'base',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_start_time (start_time),
    INDEX idx_agent_type (agent_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent 执行轨迹记录';

-- 2. Agent 执行步骤表
CREATE TABLE IF NOT EXISTS agent_steps (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    trace_id VARCHAR(255) NOT NULL,
    step_number INT NOT NULL,
    action_name VARCHAR(100) NOT NULL,
    action_args JSON,
    action_reason TEXT,
    observation_output JSON,
    observation_error TEXT,
    latency_ms BIGINT DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trace_id) REFERENCES agent_traces(id) ON DELETE CASCADE,
    INDEX idx_trace_id (trace_id),
    INDEX idx_step_number (step_number),
    INDEX idx_action_name (action_name),
    INDEX idx_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent 执行步骤详情';

-- 3. 任务管理表（多步任务管理）
CREATE TABLE IF NOT EXISTS task_plans (
    id VARCHAR(255) PRIMARY KEY,
    trace_id VARCHAR(255) NOT NULL,
    original_goal TEXT NOT NULL,
    status ENUM('pending', 'in_progress', 'completed', 'failed') NOT NULL DEFAULT 'pending',
    total_tasks INT DEFAULT 0,
    completed_tasks INT DEFAULT 0,
    failed_tasks INT DEFAULT 0,
    execution_order JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (trace_id) REFERENCES agent_traces(id) ON DELETE CASCADE,
    INDEX idx_trace_id (trace_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='任务计划管理';

-- 4. 子任务表
CREATE TABLE IF NOT EXISTS sub_tasks (
    id VARCHAR(255) PRIMARY KEY,
    plan_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    task_type ENUM('data_collection', 'content_generation', 'file_operation', 'analysis', 'other') NOT NULL,
    status ENUM('pending', 'in_progress', 'completed', 'failed', 'skipped') NOT NULL DEFAULT 'pending',
    priority INT DEFAULT 5,
    dependencies JSON,
    result JSON,
    evidence JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (plan_id) REFERENCES task_plans(id) ON DELETE CASCADE,
    INDEX idx_plan_id (plan_id),
    INDEX idx_status (status),
    INDEX idx_task_type (task_type),
    INDEX idx_priority (priority)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='子任务详情';

-- 5. MCP 服务器配置表
CREATE TABLE IF NOT EXISTS mcp_servers (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    transport ENUM('http', 'sse', 'websocket') NOT NULL DEFAULT 'http',
    url VARCHAR(500) NOT NULL,
    status ENUM('active', 'inactive', 'error') NOT NULL DEFAULT 'active',
    last_ping TIMESTAMP NULL,
    error_message TEXT,
    config JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_status (status),
    INDEX idx_transport (transport)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP 服务器配置';

-- 6. MCP 工具信息表
CREATE TABLE IF NOT EXISTS mcp_tools (
    id VARCHAR(255) PRIMARY KEY,
    server_id VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    schema JSON,
    tags JSON,
    usage_count INT DEFAULT 0,
    last_used TIMESTAMP NULL,
    avg_latency_ms BIGINT DEFAULT 0,
    success_rate DECIMAL(5,2) DEFAULT 100.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (server_id) REFERENCES mcp_servers(id) ON DELETE CASCADE,
    INDEX idx_server_id (server_id),
    INDEX idx_name (name),
    INDEX idx_usage_count (usage_count),
    INDEX idx_success_rate (success_rate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP 工具信息';

-- 7. MCP 工具调用记录表
CREATE TABLE IF NOT EXISTS mcp_tool_calls (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tool_id VARCHAR(255) NOT NULL,
    trace_id VARCHAR(255),
    args JSON,
    result JSON,
    error_message TEXT,
    latency_ms BIGINT DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tool_id) REFERENCES mcp_tools(id) ON DELETE CASCADE,
    FOREIGN KEY (trace_id) REFERENCES agent_traces(id) ON DELETE SET NULL,
    INDEX idx_tool_id (tool_id),
    INDEX idx_trace_id (trace_id),
    INDEX idx_success (success),
    INDEX idx_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP 工具调用记录';

-- 8. 系统配置表
CREATE TABLE IF NOT EXISTS system_config (
    key_name VARCHAR(100) PRIMARY KEY,
    value_data JSON,
    description TEXT,
    category VARCHAR(50) DEFAULT 'general',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置';

-- 9. 工具使用统计表
CREATE TABLE IF NOT EXISTS tool_stats (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tool_name VARCHAR(100) NOT NULL,
    tool_type ENUM('builtin', 'mcp') NOT NULL DEFAULT 'builtin',
    usage_count INT DEFAULT 1,
    total_latency_ms BIGINT DEFAULT 0,
    success_count INT DEFAULT 0,
    error_count INT DEFAULT 0,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    date_key DATE NOT NULL,
    UNIQUE KEY unique_tool_date (tool_name, tool_type, date_key),
    INDEX idx_tool_name (tool_name),
    INDEX idx_tool_type (tool_type),
    INDEX idx_date_key (date_key),
    INDEX idx_last_used (last_used)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工具使用统计';

-- 10. Agent 性能指标表
CREATE TABLE IF NOT EXISTS agent_metrics (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(10,2) NOT NULL,
    metric_unit VARCHAR(20),
    labels JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_metric_name (metric_name),
    INDEX idx_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent 性能指标';

-- 插入初始系统配置
INSERT INTO system_config (key_name, value_data, description, category) VALUES
('app_version', '"1.0.0"', 'Application version', 'system'),
('db_schema_version', '1', 'Database schema version', 'system'),
('max_trace_retention_days', '30', 'Maximum days to retain trace data', 'cleanup'),
('max_step_retention_days', '30', 'Maximum days to retain step data', 'cleanup'),
('default_agent_max_steps', '10', 'Default maximum steps for agent execution', 'agent'),
('default_agent_max_tokens', '8000', 'Default maximum tokens for agent execution', 'agent'),
('mcp_discovery_interval_minutes', '5', 'MCP tool discovery interval in minutes', 'mcp'),
('enable_metrics_collection', 'true', 'Enable performance metrics collection', 'monitoring')
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- 插入示例 MCP 服务器配置（如果需要）
-- INSERT INTO mcp_servers (id, name, transport, url, config) VALUES
-- ('stock-helper', 'Stock Helper', 'sse', 'https://mcp.example.com/stock-helper', '{"timeout": 30}'),
-- ('weather-service', 'Weather Service', 'http', 'https://api.weather.com/mcp', '{"api_key_required": true}')
-- ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- 创建用于清理旧数据的存储过程
DELIMITER //

CREATE PROCEDURE CleanupOldTraces()
BEGIN
    DECLARE retention_days INT DEFAULT 30;
    
    -- 获取配置的保留天数
    SELECT JSON_UNQUOTE(value_data) INTO retention_days 
    FROM system_config 
    WHERE key_name = 'max_trace_retention_days';
    
    -- 删除过期的轨迹数据（级联删除相关的步骤数据）
    DELETE FROM agent_traces 
    WHERE created_at < DATE_SUB(NOW(), INTERVAL retention_days DAY);
    
    -- 删除过期的 MCP 调用记录
    DELETE FROM mcp_tool_calls 
    WHERE timestamp < DATE_SUB(NOW(), INTERVAL retention_days DAY);
    
    -- 删除过期的性能指标
    DELETE FROM agent_metrics 
    WHERE timestamp < DATE_SUB(NOW(), INTERVAL retention_days DAY);
    
    -- 清理过期的统计数据
    DELETE FROM tool_stats 
    WHERE date_key < DATE_SUB(CURDATE(), INTERVAL retention_days DAY);
END //

DELIMITER ;

-- 创建用于更新工具统计的存储过程
DELIMITER //

CREATE PROCEDURE UpdateToolStats(
    IN p_tool_name VARCHAR(100),
    IN p_tool_type ENUM('builtin', 'mcp'),
    IN p_latency_ms BIGINT,
    IN p_success BOOLEAN
)
BEGIN
    INSERT INTO tool_stats (
        tool_name, tool_type, usage_count, total_latency_ms, 
        success_count, error_count, date_key
    ) VALUES (
        p_tool_name, p_tool_type, 1, p_latency_ms,
        IF(p_success, 1, 0), IF(p_success, 0, 1), CURDATE()
    )
    ON DUPLICATE KEY UPDATE
        usage_count = usage_count + 1,
        total_latency_ms = total_latency_ms + p_latency_ms,
        success_count = success_count + IF(p_success, 1, 0),
        error_count = error_count + IF(p_success, 0, 1),
        last_used = CURRENT_TIMESTAMP;
END //

DELIMITER ;

-- 创建视图以便于查询
CREATE OR REPLACE VIEW v_agent_performance AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_executions,
    AVG(duration_ms) as avg_duration_ms,
    AVG(step_count) as avg_steps,
    AVG(token_count) as avg_tokens,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as successful_executions,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_executions,
    (SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) / COUNT(*)) * 100 as success_rate
FROM agent_traces 
GROUP BY DATE(created_at)
ORDER BY date DESC;

CREATE OR REPLACE VIEW v_tool_performance AS
SELECT 
    tool_name,
    tool_type,
    SUM(usage_count) as total_usage,
    AVG(total_latency_ms / usage_count) as avg_latency_ms,
    SUM(success_count) as total_success,
    SUM(error_count) as total_errors,
    (SUM(success_count) / (SUM(success_count) + SUM(error_count))) * 100 as success_rate,
    MAX(last_used) as last_used
FROM tool_stats 
GROUP BY tool_name, tool_type
ORDER BY total_usage DESC;

CREATE OR REPLACE VIEW v_mcp_server_status AS
SELECT 
    ms.name,
    ms.transport,
    ms.url,
    ms.status,
    ms.last_ping,
    COUNT(mt.id) as tool_count,
    COALESCE(SUM(mt.usage_count), 0) as total_usage,
    COALESCE(AVG(mt.success_rate), 100) as avg_success_rate
FROM mcp_servers ms
LEFT JOIN mcp_tools mt ON ms.id = mt.server_id
GROUP BY ms.id, ms.name, ms.transport, ms.url, ms.status, ms.last_ping
ORDER BY ms.name;

-- 插入初始性能指标
INSERT INTO agent_metrics (metric_name, metric_value, metric_unit) VALUES
('system_startup', 1, 'count'),
('db_schema_initialized', 1, 'count');

-- 提交事务
COMMIT;

-- 显示初始化完成信息
SELECT 'OpenManus-Go database initialization completed successfully!' as message;