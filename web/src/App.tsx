import { Landing } from "./components/Landing";
import { Playground } from "./components/Playground";

export default function App() {
  return (
    <div className="vs-app">
      <Landing />
      <Playground />
      <footer className="vs-footer">
        <p>
          void-slice &middot; <a href="https://github.com/anthropics/void-slice">source</a>
        </p>
      </footer>
    </div>
  );
}
