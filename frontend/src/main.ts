import { createApp } from "vue";
import App from "./App.vue";
import "./style.css";

// No need to manually initialize the Wails runtime in modern versions
// Wails will automatically initialize the runtime

createApp(App).mount("#app");
