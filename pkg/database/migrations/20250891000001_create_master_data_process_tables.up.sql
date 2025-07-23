-- Create master_data_process table
CREATE TABLE IF NOT EXISTS master_data_process (
    id BIGSERIAL PRIMARY KEY,
    process_date DATE NOT NULL,
    number_of_past_days INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('RUNNING', 'COMPLETED', 'FAILED')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Create master_data_process_steps table
CREATE TABLE IF NOT EXISTS master_data_process_steps (
    id BIGSERIAL PRIMARY KEY,
    process_id BIGINT NOT NULL REFERENCES master_data_process(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL CHECK (step_number IN (1, 2, 3)),
    step_name VARCHAR(50) NOT NULL CHECK (step_name IN ('daily_ingestion', 'filter_pipeline', 'minute_ingestion')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED')),
    error_message TEXT NULL,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Create indexes for better performance
CREATE INDEX idx_master_data_process_date ON master_data_process(process_date);
CREATE INDEX idx_master_data_process_status ON master_data_process(status);
CREATE INDEX idx_master_data_process_active ON master_data_process(active);
CREATE INDEX idx_master_data_process_created_at ON master_data_process(created_at);

CREATE INDEX idx_master_data_process_steps_process_id ON master_data_process_steps(process_id);
CREATE INDEX idx_master_data_process_steps_step_number ON master_data_process_steps(step_number);
CREATE INDEX idx_master_data_process_steps_status ON master_data_process_steps(status);
CREATE INDEX idx_master_data_process_steps_active ON master_data_process_steps(active);

-- Create unique constraint to prevent duplicate processes for the same date
CREATE UNIQUE INDEX idx_master_data_process_unique_date ON master_data_process(process_date) WHERE active = TRUE;

-- Create unique constraint to prevent duplicate steps for the same process
CREATE UNIQUE INDEX idx_master_data_process_steps_unique ON master_data_process_steps(process_id, step_number) WHERE active = TRUE; 