-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL,
    auth TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    body TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(255) PRIMARY KEY,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create index on status for better query performance
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Create notification function
CREATE OR REPLACE FUNCTION notify_task_created()
    RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('tasks_channel', 
        json_build_object(
            'id', NEW.id,
            'type', NEW.type,
            'payload', NEW.payload,
            'status', NEW.status,
            'created', NEW.created,
            'updated', NEW.updated
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create notification function
CREATE OR REPLACE FUNCTION notify_notification_created()
    RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('notifications_channel', 
        json_build_object(
            'id', NEW.id,
            'body', NEW.body,
            'status', NEW.status,
            'created', NEW.created,
            'updated', NEW.updated
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS task_created_trigger ON tasks;
CREATE TRIGGER task_created_trigger
    AFTER INSERT ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION notify_task_created();

-- Create trigger
DROP TRIGGER IF EXISTS notification_created_trigger ON notifications;
CREATE TRIGGER notification_created_trigger
    AFTER INSERT ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION notify_notification_created();
