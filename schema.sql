CREATE TABLE IF NOT EXISTS wallets (
    id      UUID   PRIMARY KEY,          
    balance INT NOT NULL DEFAULT 0    
);
