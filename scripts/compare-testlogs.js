const fs = require('fs');

function compareTestResults(outputA, outputB) {
    // Split by newline, parse each JSON object, and filter out lines that don't parse
    const resultsA = outputA.split('\n').map(line => {
        try {
            return JSON.parse(line);
        } catch (e) {
            return null;
        }
    }).filter(result => result !== null);

    const resultsB = outputB.split('\n').map(line => {
        try {
            return JSON.parse(line);
        } catch (e) {
            return null;
        }
    }).filter(result => result !== null);

    // Extract test results
    const testsA = resultsA.filter(test => test.Test && ["pass", "fail"].includes(test.Action))
        .reduce((obj, test) => ({ ...obj, [test.Test]: test.Action }), {});
    const testsB = resultsB.filter(test => test.Test && ["pass", "fail"].includes(test.Action))
        .reduce((obj, test) => ({ ...obj, [test.Test]: test.Action }), {});

    // Finding tests that failed in A but not in B
    const failedInANotInB = Object.keys(testsA).filter(test => testsA[test] === "fail" && testsB[test] !== "fail");

    return failedInANotInB;
}

const filenameA = process.argv[2];
const filenameB = process.argv[3];

if (!filenameA || !filenameB) {
  console.log("Please provide filenames for both outputs as arguments.");
  process.exit(1);
}

const outputA = fs.readFileSync(filenameA, 'utf8');
const outputB = fs.readFileSync(filenameB, 'utf8');

const failedTests = compareTestResults(outputA, outputB);
console.log(`Tests failed in ${filenameA} that didn't fail in ${filenameB}:`, failedTests);
