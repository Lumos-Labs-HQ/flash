import { runGraftTests } from './graft-test';
import { runPrismaTests } from './prisma-test';
import { runDrizzleTests } from './drizzle-test';


const COLORS = {
    GRAFT: '\x1b[36m',      
    DRIZZLE: '\x1b[32m',    
    PRISMA: '\x1b[35m',     
    RESET: '\x1b[0m'
};

function drawGraph(label: string, values: number[], maxValue: number, width: number = 50) {
    const bar = (val: number, color: string) => {
        const filled = Math.max(0, Math.round((val / maxValue) * width));
        const empty = Math.max(0, width - filled);
        return color + '█'.repeat(filled) + COLORS.RESET + '░'.repeat(empty);
    };
    
    console.log(`\n${label}:`);
    console.log(`  Graft:   ${bar(values[0]!, COLORS.GRAFT)}  ${values[0]!.toFixed(0)}ms`);
    console.log(`  Drizzle: ${bar(values[1]!, COLORS.DRIZZLE)}  ${values[1]!.toFixed(0)}ms`);
    console.log(`  Prisma:  ${bar(values[2]!, COLORS.PRISMA)}  ${values[2]!.toFixed(0)}ms`);
}

async function compareBenchmarks() {
    console.log('╔══════════════════════════════════════════════════════════════════════════════╗');
    console.log('║              GRAFT vs DRIZZLE vs PRISMA - PRODUCTION BENCHMARK               ║');
    console.log('╚══════════════════════════════════════════════════════════════════════════════╝\n');

    const originalLog = console.log;
    console.log = () => {};

    const graftMetrics = await runGraftTests();
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    const drizzleMetrics = await runDrizzleTests();
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    const prismaMetrics = await runPrismaTests();

    console.log = originalLog;

    console.log('\n╔══════════════════════════════════════════════════════════════════════════════╗');
    console.log('║                           PERFORMANCE GRAPHS                                 ║');
    console.log('╚══════════════════════════════════════════════════════════════════════════════╝');

    for (let i = 0; i < graftMetrics.length; i++) {
        const times = [
            graftMetrics[i]!.totalTime,
            drizzleMetrics[i]!.totalTime,
            prismaMetrics[i]!.totalTime
        ];
        const maxTime = Math.max(...times);
        drawGraph(graftMetrics[i]!.operation, times, maxTime);
    }

    console.log('┌──────────────────────────────────────────────────┬──────────┬──────────┬──────────┐');
    console.log('│ Operation                                        │   Graft  │ Drizzle  │  Prisma  │');
    console.log('├──────────────────────────────────────────────────┼──────────┼──────────┼──────────┤');

    let totalGraft = 0, totalDrizzle = 0, totalPrisma = 0;

    for (let i = 0; i < graftMetrics.length; i++) {
        const g = graftMetrics[i]!.totalTime;
        const d = drizzleMetrics[i]!.totalTime;
        const p = prismaMetrics[i]!.totalTime;
        
        totalGraft += g;
        totalDrizzle += d;
        totalPrisma += p;

        console.log(
            `│ ${graftMetrics[i]!.operation.padEnd(48)} │ ${g.toFixed(0).padStart(6)}ms │ ${d.toFixed(0).padStart(6)}ms │ ${p.toFixed(0).padStart(6)}ms │`
        );
    }

    console.log('├──────────────────────────────────────────────────┼──────────┼──────────┼──────────┤');
    console.log(
        `│ ${'TOTAL'.padEnd(48)} │ ${totalGraft.toFixed(0).padStart(6)}ms │ ${totalDrizzle.toFixed(0).padStart(6)}ms │ ${totalPrisma.toFixed(0).padStart(6)}ms │`
    );
    console.log('└──────────────────────────────────────────────────┴──────────┴──────────┴──────────┘');

    const results = [
        { name: 'Graft', time: totalGraft, typeSafe: true },
        { name: 'Drizzle', time: totalDrizzle, typeSafe: true },
        { name: 'Prisma', time: totalPrisma, typeSafe: true }
    ];
    results.sort((a, b) => a.time - b.time);

    console.log('\n🏆 FINAL RANKINGS:');
    results.forEach((r, i) => {
        const diff = ((r.time - results[0]!.time) / results[0]!.time * 100).toFixed(1);
        const typeSafeLabel = r.typeSafe ? '✓ Type-Safe' : '✗ No Type Safety';
        console.log(`   ${i + 1}. ${r.name.padEnd(10)} - ${r.time.toFixed(0)}ms  ${typeSafeLabel}  ${i > 0 ? `(+${diff}% slower)` : '(FASTEST)'}`);
    });
}

if (import.meta.main) {
    compareBenchmarks().then(() => process.exit(0)).catch((error: any) => {
        console.error('❌ Benchmark failed:', error);
        process.exit(1);
    });
}
