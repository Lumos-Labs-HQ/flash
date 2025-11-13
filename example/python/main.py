import asyncio
import asyncpg
import os
from flash_gen.database import new

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://postgres:postgres@localhost:5432/FlashORM_test')


async def main():
    pool = await asyncpg.create_pool(DATABASE_URL)
    
    db = new(pool)
    
    newuser = await db.create_user('jack', 'jack@gmail.com', '123 street', True)
    print('New user ID:', newuser)
    
    user = await db.get_user_by_email('jack@gmail.com')
    print('User fetched by email:', user)
    
    data = await db.get_post_details_with_all_relations(1)
    print('Post details with all relations:', data)
    
    await pool.close()


if __name__ == '__main__':
    asyncio.run(main())
