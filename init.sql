-- Create tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(255) PRIMARY KEY,
    message TEXT,
    status VARCHAR(50),
    created TIMESTAMP WITH TIME ZONE
);

-- Create index on status for better query performance
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Create notification function
CREATE OR REPLACE FUNCTION notify_task_created()
    RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('tasks_channel', NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS task_created_trigger ON tasks;
CREATE TRIGGER task_created_trigger
    AFTER INSERT ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION notify_task_created();
