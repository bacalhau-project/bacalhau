/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { selection_Operator } from './selection_Operator';
export type models_LabelSelectorRequirement = {
    /**
     * key is the label key that the selector applies to.
     */
    Key?: string;
    /**
     * operator represents a key's relationship to a set of values.
     * Valid operators are In, NotIn, Exists and KeyNotInImap.
     */
    Operator?: selection_Operator;
    /**
     * values is an array of string values. If the operator is In or NotIn,
     * the values array must be non-empty. If the operator is Exists or KeyNotInImap,
     * the values array must be empty. This array is replaced during a strategic
     */
    Values?: Array<string>;
};

