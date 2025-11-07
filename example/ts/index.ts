import { Pool } from 'pg';
import { New } from './flash_gen/database';

const DATABASE_URL = process.env.DATABASE_URL || 'postgresql://postgres:postgres@localhost:5432/FlashORM_test';


async function main() {
    const pool = new Pool({
        connectionString: DATABASE_URL,
    });

    const db = New(pool);

    const newuser = await db.createUser('jack', 'jack@gmail.com', '123 street', true);
    console.log('New user ID:', newuser);

    const isAdmin = await db.isadminUser(newuser);
    if(isAdmin){
        console.log('User is admin');
    }else{
        console.log('User is not admin');
    }

    const user = await db.getUserByEmail('jack@gmail.com');
    console.log('User fetched by email:', user);

    const data = await db.getPostDetailsWithAllRelations(1);
    console.log('Post details with all relations:', data);
}


main().catch((err) => {
    console.error('Error in main:', err);
});