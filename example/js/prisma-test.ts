import { PrismaClient } from './generated/prisma/client';
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

export async function runPrismaTests() {
    console.log('\n PRISMA - PRODUCTION LOAD TEST\n');
    console.log('='.repeat(80));

    const prisma = new PrismaClient({
        datasources: {
            db: {
                url: process.env.DATABASE_URL || 'postgresql://postgres:postgres@localhost:5432/graft_test'
            }
        },
        log: ['error']
    });

    const metrics: PerformanceMetrics[] = [];

    try {
        console.log('\n Cleaning up...');
        await prisma.comments.deleteMany();
        await prisma.posts.deleteMany();
        await prisma.categories.deleteMany();
        await prisma.users.deleteMany();

        console.log('\n  Test 1: Bulk Insert - 1000 users...');
        const startMem1 = await measureMemory();
        const startTime1 = performance.now();
        const userInserts = [];
        for (let i = 1; i <= 1000; i++) {
            userInserts.push(() => prisma.users.create({
                data: {
                    name: `User ${i}`,
                    email: `user${i}@test.com`,
                    address: `Address ${i}`,
                    isadmin: i % 5 === 0
                }
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

        console.log('\n Test 2: Content creation...');
        const startMem2 = await measureMemory();
        const startTime2 = performance.now();
        const categoryInserts = [];
        for (let i = 1; i <= 10; i++) {
            categoryInserts.push(() => prisma.categories.create({ data: { name: `Category ${i}` } }));
        }
        await runConcurrent(categoryInserts, 5);
        console.log('   Creating 5000 posts...');
        const allUsers = await prisma.users.findMany({ select: { id: true } });
        const allCategories = await prisma.categories.findMany({ select: { id: true } });
        const postInserts = [];
        for (let i = 1; i <= 5000; i++) {
            postInserts.push(() => prisma.posts.create({
                data: {
                    user_id: allUsers[i % 1000]!.id,
                    category_id: allCategories[i % 10]!.id,
                    title: `Post ${i}`,
                    content: `Content ${i}...`.repeat(3)
                }
            }));
        }
        await runConcurrent(postInserts, 100);
        console.log('   Creating 15000 comments...');
        const allPosts = await prisma.posts.findMany({ select: { id: true } });
        const commentInserts = [];
        for (let i = 1; i <= 15000; i++) {
            commentInserts.push(() => prisma.comments.create({
                data: {
                    post_id: allPosts[i % 5000]!.id,
                    user_id: allUsers[i % 1000]!.id,
                    content: `Comment ${i}`
                }
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

        console.log('\n📊 Test 3: Complex queries x500 (PRISMA ORM)...');
        const startMem3 = await measureMemory();
        const startTime3 = performance.now();
        const queryTimes: number[] = [];
        for (let i = 0; i < 500; i++) {
            const qStart = performance.now();
            const users = await prisma.users.findMany({
                include: {
                    posts: {
                        select: {
                            id: true,
                            created_at: true
                        }
                    },
                    comments: {
                        select: {
                            id: true
                        }
                    }
                },
                orderBy: {
                    posts: {
                        _count: 'desc'
                    }
                }
            });
            
            const result = users.map(u => ({
                id: u.id,
                name: u.name,
                email: u.email,
                total_posts: u.posts.length,
                total_comments: u.comments.length,
                last_post_date: u.posts.length > 0 ? 
                    new Date(Math.max(...u.posts.map(p => p.created_at.getTime()))) : null
            }));
            
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

        console.log('\n⚡ Test 4: Mixed workload x1000 (PRISMA ORM)...');
        const startMem4 = await measureMemory();
        const startTime4 = performance.now();
        const mixedOps: (() => Promise<any>)[] = [];
        for (let i = 0; i < 1000; i++) {
            if (i % 4 === 0) {
                mixedOps.push(() => prisma.comments.create({
                    data: {
                        post_id: allPosts[(i % 5000)]!.id,
                        user_id: allUsers[(i % 1000)]!.id,
                        content: `RT comment ${i}`
                    }
                }));
            } else {
                if (i % 3 === 0) {
                    mixedOps.push(() => prisma.users.findMany({
                        include: {
                            posts: { select: { id: true, created_at: true } },
                            comments: { select: { id: true } }
                        }
                    }));
                } else if (i % 3 === 1) {
                    mixedOps.push(() => prisma.users.findUnique({ where: { id: allUsers[i % 1000]!.id }, select: { email: true } }));
                } else {
                    mixedOps.push(() => prisma.users.findUnique({ where: { id: allUsers[i % 1000]!.id }, select: { isadmin: true } }));
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

        console.log('\n Test 5: Stress test x2000...');
        const startMem5 = await measureMemory();
        const startTime5 = performance.now();
        const stressOps = [];
        for (let i = 0; i < 2000; i++) {
            stressOps.push(() => prisma.users.findUnique({ where: { id: allUsers[i % 1000]!.id }, select: { name: true } }));
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

        console.log('\n SUMMARY');
        console.log('='.repeat(80));
        console.log('Operation'.padEnd(45) + 'Total(ms)'.padEnd(12) + 'Avg(ms)'.padEnd(10) + 'Ops/sec');
        console.log('-'.repeat(80));
        metrics.forEach(m => {
            console.log(m.operation.padEnd(45) + m.totalTime.toFixed(2).padEnd(12) + m.avgTime.toFixed(3).padEnd(10) + (m.throughput ? m.throughput.toFixed(2) : 'N/A'));
        });

        await prisma.$disconnect();
        return metrics;
    } catch (error) {
        console.error(' Error:', error);
        await prisma.$disconnect();
        throw error;
    }
}

if (import.meta.main) {
    runPrismaTests().then(() => {
        console.log('\n Completed!\n');
        process.exit(0);
    }).catch((error: any) => {
        console.error('\n Failed:', error);
        process.exit(1);
    });
}
