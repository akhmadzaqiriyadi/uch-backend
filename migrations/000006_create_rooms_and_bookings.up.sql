-- 000006_create_rooms_and_bookings.up.sql

-- 1. Table Rooms
CREATE TABLE IF NOT EXISTS rooms (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    category VARCHAR(64) NOT NULL,
    capacity INT NOT NULL DEFAULT 1,
    location VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    facilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    image_url TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'available', -- available, maintenance, reserved
    operational_hours VARCHAR(100) DEFAULT '08:00 - 21:00 WIB',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rooms_slug ON rooms(slug);
CREATE INDEX IF NOT EXISTS idx_rooms_status ON rooms(status);

-- 2. Table Bookings
CREATE TABLE IF NOT EXISTS bookings (
    id VARCHAR(64) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id VARCHAR(64) NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    room_name VARCHAR(150) NOT NULL,
    applicant_name VARCHAR(150) NOT NULL,
    applicant_role VARCHAR(50) NOT NULL,
    id_number VARCHAR(100),
    prodi VARCHAR(150),
    purpose TEXT NOT NULL,
    audience INT NOT NULL DEFAULT 1,
    booking_date DATE NOT NULL,
    start_time VARCHAR(10) NOT NULL,
    end_time VARCHAR(10) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, cancelled, completed
    admin_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_room_id ON bookings(room_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(booking_date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);

-- 3. Seed Official UCH Rooms
INSERT INTO rooms (id, name, slug, category, capacity, location, description, facilities, image_url, status, operational_hours)
VALUES
(
    'coworking-space-hall',
    'Co-Working Space & Ideation Hall',
    'coworking-space-hall',
    'Creative & Collab Area',
    50,
    'Gedung Creative Hub Lt. 1, Kampus 1 UTY',
    'Ruang kolaborasi terbuka dan fleksibel yang dirancang untuk kerja kelompok, brainstorming ide bisnis kreatif, serta temu komunitas mahasiswa.',
    '["WiFi 6 Ultra-Fast 500 Mbps", "Smart TV 75 inch", "Standing Whiteboard", "Power Outlets Modular", "Pantry & Coffee Corner", "Ergonomic Chairs"]'::jsonb,
    '/images/coworking-space.jpg',
    'available',
    '08:00 - 21:00 WIB'
),
(
    'think-tank-meeting-room',
    'Think-Tank Meeting Room',
    'think-tank-meeting-room',
    'Meeting & Conference',
    12,
    'Gedung Creative Hub Lt. 2, Kampus 1 UTY',
    'Ruang rapat eksekutif kedap suara lengkap dengan sistem telekonferensi video 4K untuk presentasi proposal riset, rapat dosen, dan pitching mitra industri.',
    '["Soundproof Acoustic Walls", "Conference Camera 4K AI Tracking", "Interactive Smart Board 86 inch", "High-Speed LAN Port", "Central AC Climate Control", "Polycom Audio System"]'::jsonb,
    '/images/think-tank-room.jpg',
    'available',
    '08:00 - 20:00 WIB'
),
(
    'fastlab-prototyping-iot',
    'FastLab Prototyping & IoT Lab',
    'fastlab-prototyping-iot',
    'Laboratory & Hardware',
    20,
    'Gedung Creative Hub Lt. 2, Kampus 1 UTY',
    'Laboratorium fabrikasi perangkat keras, perakitan sensor IoT cerdas, pengujian mikrokontroler, dan pencetakan prototipe 3D untuk riset teknologi.',
    '["3D Printer Creality & Bambu Lab", "Oscilloscope Digital Rigol", "Soldering Station Anti-Static", "Komponen Sensor IoT & Arduino/ESP32", "Exhaust Fume Extractor", "Toolkit Mekatronika"]'::jsonb,
    '/images/prototyping-room.jpg',
    'available',
    '09:00 - 17:00 WIB'
),
(
    'multimedia-podcast-studio',
    'Studio Podcast & Multimedia',
    'multimedia-podcast-studio',
    'Audio & Broadcast',
    8,
    'Gedung Creative Hub Lt. 3, Kampus 1 UTY',
    'Studio siaran dan rekaman konten audio-visual berstandar broadcast untuk pembuatan podcast, materi kuliah digital, wawancara, dan produksi video kreatif.',
    '["Mikrofon Shure SM7B Broadcast", "Audio Interface Rodecaster Pro II", "Sony Alpha 4K Cameras (3 Angle)", "Softbox Studio Lighting Kit", "Acoustic Foam Sound Treatment", "Monitor Preview Display"]'::jsonb,
    '/images/room1.jpeg',
    'available',
    '09:00 - 18:00 WIB'
),
(
    'auditorium-pitching',
    'Auditorium Mini & Pitching Stage',
    'auditorium-pitching',
    'Event & Exhibition',
    80,
    'Gedung Creative Hub Lt. 3, Kampus 1 UTY',
    'Panggung presentasi bertingkat dengan sound system surround dan proyektor resolusi tinggi untuk demo day startup, seminar praktisi, dan pameran karya inovasi.',
    '["LED Videowall Screen 4x2 Meter", "Wireless Microphone System", "Stage Lighting Control", "Surround Sound Audio", "Podium Pembicara", "Kursi Teater Bertingkat"]'::jsonb,
    '/images/room2.jpeg',
    'available',
    '08:00 - 21:00 WIB'
),
(
    'komputasi-ai-vr',
    'Laboratorium Riset AI & VR Lab',
    'komputasi-ai-vr',
    'Laboratory & High-Tech',
    25,
    'Gedung Creative Hub Lt. 2, Kampus 1 UTY',
    'Laboratorium komputasi performa tinggi untuk pelatihan model AI, rendering grafik 3D, simulasi Metaverse, dan pengembangan aplikasi Virtual Reality.',
    '["Workstation GPU NVIDIA RTX 4090", "Meta Quest 3 VR Headsets", "High-End Mechanical Keyboards", "Fiber Optic 1 Gbps Dedicated", "Dual Monitor Ergonomic Mounts", "Server Rack GPU Compute"]'::jsonb,
    '/images/room3.jpeg',
    'available',
    '08:00 - 19:00 WIB'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    capacity = EXCLUDED.capacity,
    location = EXCLUDED.location,
    description = EXCLUDED.description,
    facilities = EXCLUDED.facilities,
    image_url = EXCLUDED.image_url,
    status = EXCLUDED.status,
    operational_hours = EXCLUDED.operational_hours;
