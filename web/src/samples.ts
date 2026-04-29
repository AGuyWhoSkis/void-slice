import countMismatch from "./samples/count-mismatch.decl?raw";
import dupIndex from "./samples/dup-index.decl?raw";
import missingSemicolon from "./samples/missing-semicolon.decl?raw";

export interface Sample {
  filename: string;
  label: string;
  hint: string;
  body: string;
}

export const SAMPLES: Sample[] = [
  {
    filename: "count-mismatch.decl",
    label: "count-mismatch",
    hint: "array num= disagrees with the number of items defined",
    body: countMismatch,
  },
  {
    filename: "dup-index.decl",
    label: "dup-index",
    hint: "two array entries share the same explicit index",
    body: dupIndex,
  },
  {
    filename: "missing-semicolon.decl",
    label: "missing-semicolon",
    hint: "assignment is not terminated with `;`",
    body: missingSemicolon,
  },
];
