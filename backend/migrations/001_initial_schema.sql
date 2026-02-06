-- Create facilities table
CREATE TABLE IF NOT EXISTS facilities (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    street VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    country VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    phone_number VARCHAR(50),
    email VARCHAR(255),
    website VARCHAR(255),
    description TEXT,
    facility_type VARCHAR(50),
    rating DECIMAL(3, 2) DEFAULT 0.0,
    review_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_facilities_location ON facilities(latitude, longitude);
CREATE INDEX idx_facilities_type ON facilities(facility_type);
CREATE INDEX idx_facilities_active ON facilities(is_active);

-- Create procedures table
CREATE TABLE IF NOT EXISTS procedures (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    category VARCHAR(100),
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_procedures_code ON procedures(code);
CREATE INDEX idx_procedures_category ON procedures(category);

-- Create facility_procedures table
CREATE TABLE IF NOT EXISTS facility_procedures (
    id VARCHAR(255) PRIMARY KEY,
    facility_id VARCHAR(255) NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    procedure_id VARCHAR(255) NOT NULL REFERENCES procedures(id) ON DELETE CASCADE,
    price DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    estimated_duration INTEGER,
    is_available BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(facility_id, procedure_id)
);

CREATE INDEX idx_facility_procedures_facility ON facility_procedures(facility_id);
CREATE INDEX idx_facility_procedures_procedure ON facility_procedures(procedure_id);
CREATE INDEX idx_facility_procedures_price ON facility_procedures(price);

-- Create insurance_providers table
CREATE TABLE IF NOT EXISTS insurance_providers (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    phone_number VARCHAR(50),
    website VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_insurance_providers_code ON insurance_providers(code);

-- Create facility_insurance table
CREATE TABLE IF NOT EXISTS facility_insurance (
    id VARCHAR(255) PRIMARY KEY,
    facility_id VARCHAR(255) NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    insurance_provider_id VARCHAR(255) NOT NULL REFERENCES insurance_providers(id) ON DELETE CASCADE,
    is_accepted BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(facility_id, insurance_provider_id)
);

CREATE INDEX idx_facility_insurance_facility ON facility_insurance(facility_id);
CREATE INDEX idx_facility_insurance_provider ON facility_insurance(insurance_provider_id);

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

-- Create appointments table
CREATE TABLE IF NOT EXISTS appointments (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE SET NULL,
    facility_id VARCHAR(255) NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    procedure_id VARCHAR(255) NOT NULL REFERENCES procedures(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    patient_name VARCHAR(255) NOT NULL,
    patient_email VARCHAR(255) NOT NULL,
    patient_phone VARCHAR(50),
    insurance_provider VARCHAR(255),
    insurance_policy_number VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_appointments_user ON appointments(user_id);
CREATE INDEX idx_appointments_facility ON appointments(facility_id);
CREATE INDEX idx_appointments_scheduled ON appointments(scheduled_at);
CREATE INDEX idx_appointments_status ON appointments(status);

-- Create availability_slots table
CREATE TABLE IF NOT EXISTS availability_slots (
    id VARCHAR(255) PRIMARY KEY,
    facility_id VARCHAR(255) NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    is_booked BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_availability_slots_facility ON availability_slots(facility_id);
CREATE INDEX idx_availability_slots_time ON availability_slots(start_time, end_time);
CREATE INDEX idx_availability_slots_booked ON availability_slots(is_booked);

-- Create reviews table
CREATE TABLE IF NOT EXISTS reviews (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    facility_id VARCHAR(255) NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, facility_id)
);

CREATE INDEX idx_reviews_user ON reviews(user_id);
CREATE INDEX idx_reviews_facility ON reviews(facility_id);
CREATE INDEX idx_reviews_rating ON reviews(rating);
