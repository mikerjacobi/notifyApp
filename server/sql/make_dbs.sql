
DROP TABLE IF EXISTS users; 
CREATE TABLE users(
    phone_number VARCHAR(10) DEFAULT "",
    name VARCHAR(255) DEFAULT "",
    birthday VARCHAR(10) DEFAULT "",
    hashword VARCHAR(255) DEFAULT "",
    verified TINYINT DEFAULT 0,
    session_id VARCHAR(36) DEFAULT "",
    created DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (phone_number),
    INDEX phone_number_index (phone_number),
    INDEX birthday_index (birthday),
    INDEX session_id_index (session_id),
    INDEX created_index (created),
    INDEX updated_index (updated)
);

DROP TABLE IF EXISTS prompts; 
CREATE TABLE prompts(
    prompt_id VARCHAR(36),
    name varchar(40),
    type varchar(40),
    template TEXT,
    created DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (prompt_id),
    INDEX created_index (created),
    INDEX updated_index (updated)
);

INSERT INTO prompts(prompt_id, name, type, template) VALUES
("0939c423-f3e2-468f-8e34-8e5b0c5391bc", "", "reminder", "hello world"),
("deaabd59-0d15-4f44-a3a8-1e3f920a3710", "register","registration", "respond with 'reg' to register!"),
("81a36dd3-8301-410c-af35-0b2a87cdd921", "register-ack","registration", "{{.Name}}, you successfully registered :)"),
("7b1ced70-a2a0-40c5-8aa5-1cc5cff3b04b", "", "question", "What did you have for lunch?");

DROP TABLE IF EXISTS user_prompts; 
CREATE TABLE user_prompts(
    prompt_id VARCHAR(36),
    phone_number VARCHAR(10),
    next_prompt_time DATETIME(6), 
    frequency VARCHAR(10),
    created DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (prompt_id, phone_number),
    INDEX phone_number_index (phone_number),
    INDEX next_prompt_time_index (next_prompt_time),
    INDEX created_index (created),
    INDEX updated_index (updated)
);

DROP TABLE IF EXISTS communications; 
CREATE TABLE communications(
    comms_id VARCHAR(36),
    from_phone VARCHAR(10),
    to_phone VARCHAR(10),
    message TEXT,
    created DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (comms_id),
    INDEX from_index (from_phone),
    INDEX to_index (to_phone),
    INDEX created_index (created)
);

DROP TABLE IF EXISTS journals; 
CREATE TABLE journals(
    journal_id VARCHAR(36),
    comms_id VARCHAR(36),
    phone_number VARCHAR(10),
    prompt TEXT,
    entry TEXT,
    created DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (journal_id),
    INDEX comms_id_index (comms_id),
    INDEX phone_number_index (phone_number),
    INDEX created_index (created),
    INDEX updated_index (updated)
);
