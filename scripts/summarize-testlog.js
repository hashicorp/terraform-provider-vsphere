/**
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

const fs = require('fs');

const filePath = process.argv[2];
if (!filePath) {
  console.error('Please provide the log file path as an argument.');
  process.exit(1);
}

const fileContent = fs.readFileSync(filePath, 'utf8');
const lines = fileContent.split('\n');

let pass = 0, skip = 0, fail = 0;

lines.forEach((line) => {
  try {
    const test = JSON.parse(line);
    if (test.Action === 'pass') pass++;
    else if (test.Action === 'skip') skip++;
    else if (test.Action === 'fail') fail++;
  } catch (err) {
    // Ignore lines that are not valid JSON
  }
});

// Print summary
console.log(`Pass: ${pass}`);
console.log(`Skip: ${skip}`);
console.log(`Fail: ${fail}`);

// Exit with code 1 if passed is 0
if (pass === 0) process.exit(1);
