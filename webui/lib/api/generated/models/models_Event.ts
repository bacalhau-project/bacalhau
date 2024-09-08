/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type models_Event = {
    /**
     * Any additional metadata that the system or user may need to know about
     * the event in order to handle it properly.
     */
    Details?: Record<string, string>;
    /**
     * A human-readable string giving the user all the information they need to
     * understand and respond to an Event, if a response is required.
     */
    Message?: string;
    /**
     * The moment the event occurred, which may be different to the moment it
     * was recorded.
     */
    Timestamp?: string;
    /**
     * The topic of the event. See the documentation on EventTopic.
     */
    Topic?: string;
};

