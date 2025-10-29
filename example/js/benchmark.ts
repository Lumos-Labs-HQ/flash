import { runGraftTests } from './graft-test';
import { runPrismaTests } from './prisma-test';

async function runComparison() {
    console.log('\n');
    console.log('═');
    console.log('          GRAFT vs PRISMA - PRODUCTION LOAD BENCHMARK');
    console.log('═');
    console.log('\n');

    try {
        console.log('Starting Graft tests...\n');
        const graftMetrics = await runGraftTests();

        console.log('\n' + '='.repeat(80));
        console.log('\nWaiting 3 seconds before Prisma tests...\n');
        await new Promise(resolve => setTimeout(resolve, 3000));

        console.log('Starting Prisma tests...\n');
        const prismaMetrics = await runPrismaTests();

        console.log('\n\n');
        console.log('');
        console.log('                            COMPARISON RESULTS');
        console.log('');
        console.log('\n');

        console.log('Operation'.padEnd(45) + 'Graft'.padEnd(15) + 'Prisma'.padEnd(15) + 'Winner');
        console.log('-'.repeat(80));

        let graftWins = 0;
        let prismaWins = 0;
        let ties = 0;

        graftMetrics.forEach((gm, idx) => {
            const pm = prismaMetrics[idx];
            if (!pm) return;

            const graftTime = gm.totalTime;
            const prismaTime = pm.totalTime;
            const diff = ((prismaTime - graftTime) / prismaTime) * 100;

            let winner = 'Tie';
            if (Math.abs(diff) > 5) {
                winner = graftTime < prismaTime ? 'Graft' : 'Prisma';
                if (winner === 'Graft') graftWins++;
                else prismaWins++;
            } else {
                ties++;
            }

            const improvement = Math.abs(diff).toFixed(1) + '%';

            console.log(
                gm.operation.padEnd(45) +
                (gm.totalTime.toFixed(2) + 'ms').padEnd(15) +
                (pm.totalTime.toFixed(2) + 'ms').padEnd(15) +
                `${winner} ${diff > 5 ? '(' + improvement + ' faster)' : ''}`
            );
        });

        console.log('\n' + '='.repeat(80));
        console.log('\nFINAL SCORE');
        console.log('-'.repeat(80));
        console.log(`Graft Wins:  ${graftWins}`);
        console.log(`Prisma Wins: ${prismaWins}`);
        console.log(`Ties:        ${ties}`);

        const totalGraftTime = graftMetrics.reduce((sum, m) => sum + m.totalTime, 0);
        const totalPrismaTime = prismaMetrics.reduce((sum, m) => sum + m.totalTime, 0);
        const overallDiff = ((totalPrismaTime - totalGraftTime) / totalPrismaTime) * 100;

        console.log('\n OVERALL:');
        console.log(`Graft Total Time:  ${totalGraftTime.toFixed(2)}ms`);
        console.log(`Prisma Total Time: ${totalPrismaTime.toFixed(2)}ms`);

        if (overallDiff > 0) {
            console.log(`\n GRAFT IS ${overallDiff.toFixed(1)}% FASTER OVERALL!`);
        } else {
            console.log(`\n PRISMA IS ${Math.abs(overallDiff).toFixed(1)}% FASTER OVERALL!`);
        }

        console.log('\n═\n');

    } catch (error) {
        console.error(' Error running benchmark:', error);
        process.exit(1);
    }
}

if (import.meta.main) {
    runComparison().then(() => {
        process.exit(0);
    }).catch((error: any) => {
        console.error('Benchmark failed:', error);
        process.exit(1);
    });
}
