const fs = require('fs');
const code = fs.readFileSync('frontend/h5/app.js', 'utf-8');
const lines = code.split('\n');

let state = 'code';
let templateStack = [];
let currentExprDepth = 0;

for (let i = 0; i < code.length; i++) {
  const ch = code[i];
  
  if (state === 'code') {
    if (ch === '`') {
      state = 'template';
      templateStack.push(code.substring(0, i).split('\n').length);
      currentExprDepth = 0;
    }
  } else if (state === 'template') {
    if (ch === '`') {
      if (currentExprDepth === 0) {
        state = 'code';
        templateStack.pop();
      } else {
        templateStack.push(code.substring(0, i).split('\n').length);
      }
    } else if (ch === '$' && i+1 < code.length && code[i+1] === '{') {
      currentExprDepth++;
    } else if (ch === '}' && currentExprDepth > 0) {
      currentExprDepth--;
    }
  }
}

if (templateStack.length > 0) {
  console.log('UNCLOSED template literal(s):');
  for (const ln of templateStack) {
    console.log('  Line ' + ln + ': ' + lines[ln-1].substring(0, 120));
  }
} else {
  console.log('All template literals properly closed');
}
