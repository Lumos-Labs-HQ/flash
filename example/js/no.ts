import { Pool } from 'pg';
import { New } from './graft_gen/database';
import type { Users } from './graft_gen';

const DATABASE_URL = process.env.DATABASE_URL || 'postgresql://postgres:postgres@localhost:5432/graft_test';

const pool = new Pool({
    connectionString: DATABASE_URL,
    max: 20,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 2000,
});
const db = New(pool);

async function main() {

    const exists = await db.checkUserExists('jackc@gmail.com');
    if (!exists) {
        await db.createUser('jack', 'jackc@gmail.com', 'my address', true);
        console.log('Created user');
    } else {
        const user = await db.getUserByEmail('jackc@gmail.com')
        console.log('User already exists:', user?.email);
    }
    pool.end();
}

main().catch(err => {
    console.error('Error running example:', err);
    pool.end();
});
