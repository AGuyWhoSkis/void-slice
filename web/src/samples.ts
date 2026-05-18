import clean from "./samples/clean.decl?raw";
import indexOob from "./samples/index-oob.decl?raw";
import dupIndex from "./samples/dup-index.decl?raw";
import missingSemicolon from "./samples/missing-semicolon.decl?raw";

export interface Sample {
  filename: string;
  label: string;
  hint: string;
  body: string;
  clean?: boolean;
}

export const SAMPLES: Sample[] = [
  {
    filename: "clean.decl",
    label: "clean — try breaking it",
    hint: "a known-good file: flip a quote, delete a semicolon, change num=, and watch the linter catch you",
    body: clean,
    clean: true,
  },
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
