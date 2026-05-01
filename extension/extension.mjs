import { joinSession } from "@github/copilot-sdk/extension";
import { createSessionConfig } from "./session-config.mjs";

await joinSession(createSessionConfig());
