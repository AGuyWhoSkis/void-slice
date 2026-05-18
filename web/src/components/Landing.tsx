import { ThemeToggle } from "./ThemeToggle";

export function Landing() {
  return (
    <section className="vs-landing">
      <div className="vs-landing-toggle">
        <ThemeToggle />
      </div>
      <div className="vs-landing-inner">
        <h1>void-slice</h1>
        <p className="vs-tagline">
          A linter for Dishonored 2 and Death of the Outsider game files. Spot
          structural problems in <code>.decl</code>, <code>.entities</code>, and{" "}
          <code>.entitydef</code> files before they break a level.
        </p>
        <p>
          Paste, drop, or click a sample below. Then try breaking it — flip a
          quote, delete a semicolon, change a <code>num=</code> — and watch the
          diagnostics update as you type. Same engine the CLI uses.
        </p>
        <a className="vs-cta" href="#playground">
          Try it →
        </a>
      </div>
    </section>
  );
}
