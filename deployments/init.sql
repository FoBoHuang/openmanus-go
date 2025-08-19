-- OpenManus-Go 数据库初始化脚本

-- 创建示例表
CREATE TABLE IF NOT EXISTS tasks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    goal TEXT NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending',
    agent_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    result TEXT,
    metadata JSON
);

-- 创建执行轨迹表
CREATE TABLE IF NOT EXISTS traces (
    id VARCHAR(255) PRIMARY KEY,
    goal TEXT NOT NULL,
    status ENUM('running', 'completed', 'failed', 'canceled') DEFAULT 'running',
    steps_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    trace_data JSON
);

-- 创建工具使用统计表
CREATE TABLE IF NOT EXISTS tool_usage (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tool_name VARCHAR(255) NOT NULL,
    usage_count INT DEFAULT 1,
    success_count INT DEFAULT 0,
    failure_count INT DEFAULT 0,
    avg_latency_ms INT DEFAULT 0,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_tool (tool_name)
);

-- 插入一些示例数据
INSERT INTO tasks (goal, status, agent_id, result) VALUES
('创建一个简单的 Hello World 文件', 'completed', 'agent-001', 'Successfully created hello.txt file'),
('分析 CSV 数据并生成报告', 'completed', 'agent-002', 'Data analysis completed, report generated'),
('搜索最新的技术新闻', 'pending', NULL, NULL);

-- 插入工具使用统计
INSERT INTO tool_usage (tool_name, usage_count, success_count, failure_count, avg_latency_ms) VALUES
('http', 15, 12, 3, 1200),
('fs', 25, 24, 1, 50),
('redis', 8, 8, 0, 30),
('mysql', 5, 4, 1, 80),
('browser', 3, 2, 1, 5000),
('crawler', 7, 6, 1, 3000);

-- 创建索引以提高查询性能
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
CREATE INDEX idx_traces_status ON traces(status);
CREATE INDEX idx_traces_created_at ON traces(created_at);
CREATE INDEX idx_tool_usage_tool_name ON tool_usage(tool_name);

-- 创建视图：任务统计
CREATE VIEW task_statistics AS
SELECT 
    status,
    COUNT(*) as count,
    AVG(TIMESTAMPDIFF(SECOND, created_at, updated_at)) as avg_duration_seconds
FROM tasks 
GROUP BY status;

-- 创建视图：工具性能统计
CREATE VIEW tool_performance AS
SELECT 
    tool_name,
    usage_count,
    success_count,
    failure_count,
    ROUND((success_count / usage_count) * 100, 2) as success_rate_percent,
    avg_latency_ms
FROM tool_usage
ORDER BY usage_count DESC;
