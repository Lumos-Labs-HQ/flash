import { runGraftTests } from './graft-test';
import { runPrismaTests } from './prisma-test';
import { runDrizzleTests } from './drizzle-test';

async function compareBenchmarks() {
    console.log('╔══════════════════════════════════════════════════════════════════════════════╗');
    console.log('║            GRAFT vs DRIZZLE vs PRISMA - PRODUCTION BENCHMARK                 ║');
    console.log('╚══════════════════════════════════════════════════════════════════════════════╝\n');

    try {
        const graftMetrics = await runGraftTests();
        
        console.log('\n⏳ Waiting 3 seconds before next test...\n');
        await new Promise(resolve => setTimeout(resolve, 3000));
        
        const drizzleMetrics = await runDrizzleTests();
        
        console.log('\n⏳ Waiting 3 seconds before next test...\n');
        await new Promise(resolve => setTimeout(resolve, 3000));
        
        const prismaMetrics = await runPrismaTests();

        console.log('\n╔══════════════════════════════════════════════════════════════════════════════╗');
        console.log('║                        3-WAY PERFORMANCE COMPARISON                          ║');
        console.log('╚══════════════════════════════════════════════════════════════════════════════╝\n');

        console.log('Operation'.padEnd(50) + 'Graft(ms)'.padEnd(13) + 'Drizzle(ms)'.padEnd(15) + 'Prisma(ms)'.padEnd(13) + 'Winner');
        console.log('='.repeat(105));

        let graftWins = 0;
        let drizzleWins = 0;
        let prismaWins = 0;
        let totalGraftTime = 0;
        let totalDrizzleTime = 0;
        let totalPrismaTime = 0;

        for (let i = 0; i < graftMetrics.length; i++) {
            const graft = graftMetrics[i]!;
            const drizzle = drizzleMetrics[i]!;
            const prisma = prismaMetrics[i]!;
            
            totalGraftTime += graft.totalTime;
            totalDrizzleTime += drizzle.totalTime;
            totalPrismaTime += prisma.totalTime;

            const minTime = Math.min(graft.totalTime, drizzle.totalTime, prisma.totalTime);
            let winner = '';
            
            if (graft.totalTime === minTime) {
                graftWins++;
                const drizzleDiff = ((drizzle.totalTime - graft.totalTime) / graft.totalTime * 100).toFixed(1);
                const prismaDiff = ((prisma.totalTime - graft.totalTime) / graft.totalTime * 100).toFixed(1);
                winner = `🏆 GRAFT (+${drizzleDiff}% vs Drizzle, +${prismaDiff}% vs Prisma)`;
            } else if (drizzle.totalTime === minTime) {
                drizzleWins++;
                const graftDiff = ((graft.totalTime - drizzle.totalTime) / drizzle.totalTime * 100).toFixed(1);
                const prismaDiff = ((prisma.totalTime - drizzle.totalTime) / drizzle.totalTime * 100).toFixed(1);
                winner = `🏆 DRIZZLE (+${graftDiff}% vs Graft, +${prismaDiff}% vs Prisma)`;
            } else {
                prismaWins++;
                const graftDiff = ((graft.totalTime - prisma.totalTime) / prisma.totalTime * 100).toFixed(1);
                const drizzleDiff = ((drizzle.totalTime - prisma.totalTime) / prisma.totalTime * 100).toFixed(1);
                winner = `🏆 PRISMA (+${graftDiff}% vs Graft, +${drizzleDiff}% vs Drizzle)`;
            }

            console.log(
                graft.operation.padEnd(50) + 
                graft.totalTime.toFixed(2).padEnd(13) + 
                drizzle.totalTime.toFixed(2).padEnd(15) + 
                prisma.totalTime.toFixed(2).padEnd(13) + 
                winner
            );
        }

        console.log('='.repeat(105));
        console.log('TOTAL'.padEnd(50) + 
                    totalGraftTime.toFixed(2).padEnd(13) + 
                    totalDrizzleTime.toFixed(2).padEnd(15) + 
                    totalPrismaTime.toFixed(2).padEnd(13));
        
        console.log('\n📊 FINAL SCORE:');
        console.log(`   Graft wins: ${graftWins}/5`);
        console.log(`   Drizzle wins: ${drizzleWins}/5`);
        console.log(`   Prisma wins: ${prismaWins}/5`);
        
        const minTotalTime = Math.min(totalGraftTime, totalDrizzleTime, totalPrismaTime);
        if (totalGraftTime === minTotalTime) {
            const drizzleDiff = ((totalDrizzleTime - totalGraftTime) / totalGraftTime * 100).toFixed(1);
            const prismaDiff = ((totalPrismaTime - totalGraftTime) / totalGraftTime * 100).toFixed(1);
            console.log(`\n🏆 OVERALL WINNER: GRAFT (${drizzleDiff}% faster than Drizzle, ${prismaDiff}% faster than Prisma)`);
        } else if (totalDrizzleTime === minTotalTime) {
            const graftDiff = ((totalGraftTime - totalDrizzleTime) / totalDrizzleTime * 100).toFixed(1);
            const prismaDiff = ((totalPrismaTime - totalDrizzleTime) / totalDrizzleTime * 100).toFixed(1);
            console.log(`\n🏆 OVERALL WINNER: DRIZZLE (${graftDiff}% faster than Graft, ${prismaDiff}% faster than Prisma)`);
        } else {
            const graftDiff = ((totalGraftTime - totalPrismaTime) / totalPrismaTime * 100).toFixed(1);
            const drizzleDiff = ((totalDrizzleTime - totalPrismaTime) / totalPrismaTime * 100).toFixed(1);
            console.log(`\n🏆 OVERALL WINNER: PRISMA (${graftDiff}% faster than Graft, ${drizzleDiff}% faster than Drizzle)`);
        }

    } catch (error) {
        console.error('❌ Error running benchmark:', error);
        process.exit(1);
    }
}

if (import.meta.main) {
    compareBenchmarks().then(() => {
        console.log('\n✅ Benchmark completed!\n');
        process.exit(0);
    }).catch((error: any) => {
        console.error('❌ Benchmark failed:', error);
        process.exit(1);
    });
}
