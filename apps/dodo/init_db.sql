CREATE TABLE waitlist (
	   id INTEGER NOT NULL PRIMARY KEY,
	   timestamp DEFAULT CURRENT_TIMESTAMP,
	   email VARCHAR(255),
	   installation_type VARCHAR(255),
	   num_members INTEGER,
	   apps VARCHAR(10000),
	   pay_per_month DOUBLE,
	   prepay_full_year BOOLEAN,
	   thoughts VARCHAR(10000)
);
