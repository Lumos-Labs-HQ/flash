import { Pool } from 'pg';
import { New } from './graft_gen/database';

const DATABASE_URL = process.env.DATABASE_URL || 'postgresql://postgres:postgres@localhost:5432/graft_test';

export interface PerformanceMetrics {
    operation: string;
    totalTime: number;
    avgTime: number;
    minTime?: number;
    maxTime?: number;
    memoryUsed: number;
    peakMemory: number;
    throughput?: number;
    concurrentOps?: number;
}

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

export async function runGraftTests() {
    console.log('\n🔧 GRAFT - PRODUCTION LOAD TEST\n');
    console.log('='.repeat(80));

    const pool = new Pool({
        connectionString: DATABASE_URL,
        max: 30,
        idleTimeoutMillis: 30000,
        connectionTimeoutMillis: 2000,
    });
    const db = New(pool);
    const metrics: PerformanceMetrics[] = [];

    try {
        console.log('\n Cleaning up...');
        await pool.query('TRUNCATE users, posts, categories, comments RESTART IDENTITY CASCADE');

        console.log('\n✍️  Test 1: Bulk Insert - 1000 users (OPTIMIZED)...');
        const startMem1 = await measureMemory();
        const startTime1 = performance.now();
        const userInserts = [];
        for (let i = 1; i <= 1000; i++) {
            userInserts.push(() => db.createUser(`User ${i}`, `user${i}@test.com`, `Address ${i}`, i % 5 === 0));
        }
        await runConcurrent(userInserts, 100);
        const endTime1 = performance.now();
        const endMem1 = await measureMemory();
        metrics.push({
            operation: 'Insert 1000 Users',
            totalTime: endTime1 - startTime1,
            avgTime: (endTime1 - startTime1) / 1000,
            memoryUsed: endMem1 - startMem1,
            peakMemory: endMem1,
            throughput: 1000 / ((endTime1 - startTime1) / 1000),
            concurrentOps: 100
        });
        console.log(`   Time: ${(endTime1 - startTime1).toFixed(2)}ms | Throughput: ${(1000 / ((endTime1 - startTime1) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n📂 Test 2: Content creation (OPTIMIZED)...');
        const startMem2 = await measureMemory();
        const startTime2 = performance.now();
        const categoryInserts = [];
        for (let i = 1; i <= 10; i++) {
            categoryInserts.push(() => db.createCategory(`Category ${i}`));
        }
        await runConcurrent(categoryInserts, 10);
        console.log('   Creating 5000 posts...');
        const postInserts = [];
        for (let i = 1; i <= 5000; i++) {
            postInserts.push(() => db.createPost((i % 1000) + 1, (i % 10) + 1, `Post ${i}`, `Content ${i}...`.repeat(3)));
        }
        await runConcurrent(postInserts, 150);
        console.log('   Creating 15000 comments...');
        const commentInserts = [];
        for (let i = 1; i <= 15000; i++) {
            commentInserts.push(() => db.createComment((i % 5000) + 1, (i % 1000) + 1, `Comment ${i}`));
        }
        await runConcurrent(commentInserts, 150);
        const endTime2 = performance.now();
        const endMem2 = await measureMemory();
        metrics.push({
            operation: 'Insert 10 Cat + 5K Posts + 15K Comments',
            totalTime: endTime2 - startTime2,
            avgTime: (endTime2 - startTime2) / 20010,
            memoryUsed: endMem2 - startMem2,
            peakMemory: endMem2,
            throughput: 20010 / ((endTime2 - startTime2) / 1000),
            concurrentOps: 150
        });
        console.log(`   Time: ${(endTime2 - startTime2).toFixed(2)}ms | Throughput: ${(20010 / ((endTime2 - startTime2) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n🔍 Test 3: Complex queries x500 (GRAFT GENERATED)...');
        const startMem3 = await measureMemory();
        const startTime3 = performance.now();
        const queryTimes: number[] = [];

        const queryOps = [];
        for (let i = 0; i < 500; i++) {
            queryOps.push(async () => {
                const qStart = performance.now();
                await db.getActiveUsersWithStats();
                queryTimes.push(performance.now() - qStart);
            });
        }
        await runConcurrent(queryOps, 5);
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

        console.log('\n⚡ Test 4: Mixed workload x1000 (OPTIMIZED)...');
        const startMem4 = await measureMemory();
        const startTime4 = performance.now();
        const mixedOps: (() => Promise<any>)[] = [];
        for (let i = 0; i < 1000; i++) {
            if (i % 4 === 0) {
                mixedOps.push(() => db.createComment((i % 5000) + 1, (i % 1000) + 1, `RT comment ${i}`));
            } else {
                if (i % 3 === 0) mixedOps.push(() => db.getActiveUsersWithStats());
                else if (i % 3 === 1) mixedOps.push(() => db.getUserEmail((i % 1000) + 1));
                else mixedOps.push(() => db.isadminUser((i % 1000) + 1));
            }
        }
        await runConcurrent(mixedOps, 50);
        const endTime4 = performance.now();
        const endMem4 = await measureMemory();
        metrics.push({
            operation: 'Mixed Workload x1000 (75% read, 25% write)',
            totalTime: endTime4 - startTime4,
            avgTime: (endTime4 - startTime4) / 1000,
            memoryUsed: endMem4 - startMem4,
            peakMemory: endMem4,
            throughput: 1000 / ((endTime4 - startTime4) / 1000),
            concurrentOps: 50
        });
        console.log(`   Time: ${(endTime4 - startTime4).toFixed(2)}ms | Throughput: ${(1000 / ((endTime4 - startTime4) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n🔥 Test 5: Stress test x2000 (OPTIMIZED)...');
        const startMem5 = await measureMemory();
        const startTime5 = performance.now();
        const stressOps = [];
        for (let i = 0; i < 2000; i++) {
            stressOps.push(() => db.getUserName((i % 1000) + 1));
        }
        await runConcurrent(stressOps, 100);
        const endTime5 = performance.now();
        const endMem5 = await measureMemory();
        metrics.push({
            operation: 'Stress Test - Simple Query x2000',
            totalTime: endTime5 - startTime5,
            avgTime: (endTime5 - startTime5) / 2000,
            memoryUsed: endMem5 - startMem5,
            peakMemory: endMem5,
            throughput: 2000 / ((endTime5 - startTime5) / 1000),
            concurrentOps: 100
        });
        console.log(`   Time: ${(endTime5 - startTime5).toFixed(2)}ms | Throughput: ${(2000 / ((endTime5 - startTime5) / 1000)).toFixed(2)} ops/sec`);

        console.log('\n Pool: Total=' + pool.totalCount + ' Idle=' + pool.idleCount + ' Waiting=' + pool.waitingCount);
        console.log('\n SUMMARY');
        console.log('='.repeat(80));
        console.log('Operation'.padEnd(45) + 'Total(ms)'.padEnd(12) + 'Avg(ms)'.padEnd(10) + 'Ops/sec');
        console.log('-'.repeat(80));
        metrics.forEach(m => {
            console.log(m.operation.padEnd(45) + m.totalTime.toFixed(2).padEnd(12) + m.avgTime.toFixed(3).padEnd(10) + (m.throughput ? m.throughput.toFixed(2) : 'N/A'));
        });

        await pool.end();
        return metrics;
    } catch (error) {
        console.error(' Error:', error);
        await pool.end();
        throw error;
    }
}

if (import.meta.main) {
    runGraftTests().then(() => {
        console.log('\n Completed!\n');
        process.exit(0);
    }).catch((error: any) => {
        console.error('\n Failed:', error);
        process.exit(1);
    });
}
