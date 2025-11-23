CREATE TABLE IF NOT EXISTS teams (
    team_id SERIAL PRIMARY KEY,
    team_name VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    username VARCHAR(100) NOT NULL,          
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id INT REFERENCES teams(team_id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(user_id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id TEXT PRIMARY KEY, 
    pull_request_name VARCHAR(200) NOT NULL,
    author_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE TABLE IF NOT EXISTS reviewers (
    pull_request_id TEXT REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(user_id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (pull_request_id, user_id)
);