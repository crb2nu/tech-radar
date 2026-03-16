-- Tech Radar Enterprise Schema

CREATE TABLE IF NOT EXISTS radars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    version TEXT,
    config JSONB
);

CREATE TABLE IF NOT EXISTS entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    radar_id UUID REFERENCES radars(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    quadrant TEXT NOT NULL,
    ring TEXT NOT NULL,
    description TEXT,
    moved INTEGER DEFAULT 0,
    link TEXT,
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS entry_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID REFERENCES entries(id) ON DELETE CASCADE,
    date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ring TEXT NOT NULL,
    reason TEXT,
    author TEXT
);

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    url TEXT,
    last_scan TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS project_technologies (
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    tech_name TEXT NOT NULL,
    version TEXT,
    source TEXT,
    PRIMARY KEY (project_id, tech_name)
);

CREATE TABLE IF NOT EXISTS entry_projects (
    entry_id UUID REFERENCES entries(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, project_id)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast tech lookup
CREATE INDEX IF NOT EXISTS idx_entries_name ON entries(name);
CREATE INDEX IF NOT EXISTS idx_project_technologies_name ON project_technologies(tech_name);
