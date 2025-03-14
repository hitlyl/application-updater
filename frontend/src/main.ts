import { createApp } from "vue";
import App from "./App.vue";
import "./style.css";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";

// No need to manually initialize the Wails runtime in modern versions
// Wails will automatically initialize the runtime

const app = createApp(App);
app.use(ElementPlus);
app.mount("#app");
