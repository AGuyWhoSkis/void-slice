import indexOob from "./samples/index-oob.decl?raw";
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
    filename: "index-oob.decl",
    label: "index-oob",
    hint: "array index is outside the declared num= range",
    body: indexOob,
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
