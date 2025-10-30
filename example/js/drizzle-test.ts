import { drizzle } from 'drizzle-orm/node-postgres';
import { Pool } from 'pg';
import { eq, sql, desc, count, max } from 'drizzle-orm';
import { users, categories, posts, comments } from './drizzle/schema';
import type { PerformanceMetrics } from './graft-test';

async function measureMemory(): Promise<number> {
    const used = process.memoryUsage();
    return used.heapUsed / 1024 / 1024;
}

async function runConcurrent<T>(operations: (() => Promise<T>)[], batchSize: number = 10): Promise<T[]> {
    const results: T[] = [];
    for (let i = 0; i < operations.length; i += batchSize) {
        const batch = operations.slice(i, i + batchSize);
        const batchResults = await Promise.all(batch.map(op => op()));
        results.push(...batchResults);
    }
    return results;
}

export async function runDrizzleTests() {
    console.log('\n🐉 DRIZZLE - PRODUCTION LOAD TEST\n');
    console.log('='.repeat(80));

    const pool = new Pool({ 
        connectionString: process.env.DATABASE_URL || 'postgresql://postgres:postgres@localhost:5432/graft_test',
        max: 20 
    });
    const db = drizzle(pool);
    const metrics: PerformanceMetrics[] = [];

    try {
        console.log('\n📦 Cleaning up...');
        await db.delete(comments);
        await db.delete(posts);
        await db.delete(categories);
        await db.delete(users);

        console.log('\n✍️  Test 1: Bulk Insert - 1000 users...');
        const startMem1 = await measureMemory();
        const startTime1 = performance.now();
        const userInserts = [];
        for (let i = 1; i <= 1000; i++) {
            userInserts.push(() => db.insert(users).values({
                name: `User ${i}`,
                email: `user${i}@test.com`,
                address: `Address ${i}`,
                isadmin: i % 5 === 0
            }));
        }
        await runConcurrent(userInserts, 50);
        const endTime1 = performance.now();
        const endMem1 = await measureMemory();
        metrics.push({
            operation: 'Insert 1000 Users',
            totalTime: endTime1 - startTime1,
            avgTime: (endTime1 - startTime1) / 1000,
            memoryUsed: endMem1 - startMem1,
            peakMemory: endMem1,
            throughput: 1000 / ((endTime1 - startTime1) / 1000),
            concurrentOps: 50
        });
        console.log(`   Time: ${(endTime1 - startTime1).toFixed(2)}ms | Throughput: ${(1000 / ((endTime1 - startTime1) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n📂 Test 2: Content creation...');
        const startMem2 = await measureMemory();
        const startTime2 = performance.now();
        const categoryInserts = [];
        for (let i = 1; i <= 10; i++) {
            categoryInserts.push(() => db.insert(categories).values({ name: `Category ${i}` }));
        }
        await runConcurrent(categoryInserts, 5);
        
        console.log('   Creating 5000 posts...');
        const allUsers = await db.select({ id: users.id }).from(users);
        const allCategories = await db.select({ id: categories.id }).from(categories);
        const postInserts = [];
        for (let i = 1; i <= 5000; i++) {
            postInserts.push(() => db.insert(posts).values({
                user_id: allUsers[i % 1000]!.id,
                category_id: allCategories[i % 10]!.id,
                title: `Post ${i}`,
                content: `Content ${i}...`.repeat(3)
            }));
        }
        await runConcurrent(postInserts, 100);
        
        console.log('   Creating 15000 comments...');
        const allPosts = await db.select({ id: posts.id }).from(posts);
        const commentInserts = [];
        for (let i = 1; i <= 15000; i++) {
            commentInserts.push(() => db.insert(comments).values({
                post_id: allPosts[i % 5000]!.id,
                user_id: allUsers[i % 1000]!.id,
                content: `Comment ${i}`
            }));
        }
        await runConcurrent(commentInserts, 100);
        const endTime2 = performance.now();
        const endMem2 = await measureMemory();
        metrics.push({
            operation: 'Insert 10 Cat + 5K Posts + 15K Comments',
            totalTime: endTime2 - startTime2,
            avgTime: (endTime2 - startTime2) / 20010,
            memoryUsed: endMem2 - startMem2,
            peakMemory: endMem2,
            throughput: 20010 / ((endTime2 - startTime2) / 1000),
            concurrentOps: 100
        });
        console.log(`   Time: ${(endTime2 - startTime2).toFixed(2)}ms | Throughput: ${(20010 / ((endTime2 - startTime2) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n🔍 Test 3: Complex queries x500...');
        const startMem3 = await measureMemory();
        const startTime3 = performance.now();
        const queryTimes: number[] = [];
        for (let i = 0; i < 500; i++) {
            const qStart = performance.now();
            await db
                .select({
                    id: users.id,
                    name: users.name,
                    email: users.email,
                    total_posts: count(posts.id).as('total_posts'),
                    total_comments: count(comments.id).as('total_comments'),
                    last_post_date: max(posts.created_at).as('last_post_date')
                })
                .from(users)
                .leftJoin(posts, eq(users.id, posts.user_id))
                .leftJoin(comments, eq(users.id, comments.user_id))
                .groupBy(users.id, users.name, users.email)
                .orderBy(desc(count(posts.id)));
            queryTimes.push(performance.now() - qStart);
        }
        const endTime3 = performance.now();
        const endMem3 = await measureMemory();
        metrics.push({
            operation: 'Complex Query x500',
            totalTime: endTime3 - startTime3,
            avgTime: queryTimes.reduce((a, b) => a + b, 0) / queryTimes.length,
            minTime: Math.min(...queryTimes),
            maxTime: Math.max(...queryTimes),
            memoryUsed: endMem3 - startMem3,
            peakMemory: endMem3,
            throughput: 500 / ((endTime3 - startTime3) / 1000)
        });
        console.log(`   Time: ${(endTime3 - startTime3).toFixed(2)}ms | Avg: ${(queryTimes.reduce((a, b) => a + b, 0) / queryTimes.length).toFixed(2)}ms | Min/Max: ${Math.min(...queryTimes).toFixed(2)}/${Math.max(...queryTimes).toFixed(2)}ms`);

        console.log('\n⚡ Test 4: Mixed workload x1000...');
        const startMem4 = await measureMemory();
        const startTime4 = performance.now();
        const mixedOps: (() => Promise<any>)[] = [];
        for (let i = 0; i < 1000; i++) {
            if (i % 4 === 0) {
                mixedOps.push(() => db.insert(comments).values({
                    post_id: allPosts[i % 5000]!.id,
                    user_id: allUsers[i % 1000]!.id,
                    content: `RT comment ${i}`
                }));
            } else {
                if (i % 3 === 0) {
                    mixedOps.push(() => db
                        .select({
                            id: users.id,
                            name: users.name,
                            email: users.email,
                            total_posts: count(posts.id),
                            total_comments: count(comments.id),
                        })
                        .from(users)
                        .leftJoin(posts, eq(users.id, posts.user_id))
                        .leftJoin(comments, eq(users.id, comments.user_id))
                        .groupBy(users.id)
                        .limit(10));
                } else if (i % 3 === 1) {
                    mixedOps.push(() => db.select({ email: users.email }).from(users).where(eq(users.id, allUsers[i % 1000]!.id)));
                } else {
                    mixedOps.push(() => db.select({ isadmin: users.isadmin }).from(users).where(eq(users.id, allUsers[i % 1000]!.id)));
                }
            }
        }
        await runConcurrent(mixedOps, 30);
        const endTime4 = performance.now();
        const endMem4 = await measureMemory();
        metrics.push({
            operation: 'Mixed Workload x1000 (75% read, 25% write)',
            totalTime: endTime4 - startTime4,
            avgTime: (endTime4 - startTime4) / 1000,
            memoryUsed: endMem4 - startMem4,
            peakMemory: endMem4,
            throughput: 1000 / ((endTime4 - startTime4) / 1000),
            concurrentOps: 30
        });
        console.log(`   Time: ${(endTime4 - startTime4).toFixed(2)}ms | Throughput: ${(1000 / ((endTime4 - startTime4) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n🔥 Test 5: Stress test x2000...');
        const startMem5 = await measureMemory();
        const startTime5 = performance.now();
        const stressOps = [];
        for (let i = 0; i < 2000; i++) {
            stressOps.push(() => db.select({ name: users.name }).from(users).where(eq(users.id, allUsers[i % 1000]!.id)));
        }
        await runConcurrent(stressOps, 50);
        const endTime5 = performance.now();
        const endMem5 = await measureMemory();
        metrics.push({
            operation: 'Stress Test - Simple Query x2000',
            totalTime: endTime5 - startTime5,
            avgTime: (endTime5 - startTime5) / 2000,
            memoryUsed: endMem5 - startMem5,
            peakMemory: endMem5,
            throughput: 2000 / ((endTime5 - startTime5) / 1000),
            concurrentOps: 50
        });
        console.log(`   Time: ${(endTime5 - startTime5).toFixed(2)}ms | Throughput: ${(2000 / ((endTime5 - startTime5) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n📊 SUMMARY');
        console.log('='.repeat(80));
        console.log('Operation'.padEnd(45) + 'Total(ms)'.padEnd(12) + 'Avg(ms)'.padEnd(10) + 'Ops/sec');
        console.log('-'.repeat(80));
        metrics.forEach(m => {
            console.log(m.operation.padEnd(45) + m.totalTime.toFixed(2).padEnd(12) + m.avgTime.toFixed(3).padEnd(10) + (m.throughput ? m.throughput.toFixed(2) : 'N/A'));
        });

        await pool.end();
        return metrics;
    } catch (error) {
        console.error('❌ Error:', error);
        await pool.end();
        throw error;
    }
}

if (import.meta.main) {
    runDrizzleTests().then(() => {
        console.log('\n✅ Completed!\n');
        process.exit(0);
    }).catch((error: any) => {
        console.error('\n❌ Failed:', error);
        process.exit(1);
    });
}
