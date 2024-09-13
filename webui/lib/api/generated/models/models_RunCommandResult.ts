/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type models_RunCommandResult = {
    /**
     * Runner error
     */
    ErrorMsg?: string;
    /**
     * exit code of the run.
     */
    ExitCode?: number;
    /**
     * bool describing if stderr was truncated
     */
    StderrTruncated?: boolean;
    /**
     * stdout of the run. Yaml provided for `describe` output
     */
    Stdout?: string;
    /**
     * bool describing if stdout was truncated
     */
    StdoutTruncated?: boolean;
    /**
     * stderr of the run.
     */
    stderr?: string;
};

