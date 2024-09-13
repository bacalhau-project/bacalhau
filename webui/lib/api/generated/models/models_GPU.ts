/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_GPUVendor } from './models_GPUVendor';
export type models_GPU = {
    /**
     * Self-reported index of the device in the system
     */
    index?: number;
    /**
     * Total GPU memory in mebibytes (MiB)
     */
    memory?: number;
    /**
     * Model name of the GPU e.g. Tesla T4
     */
    name?: string;
    /**
     * PCI address of the device, in the format AAAA:BB:CC.C
     * Used to discover the correct device rendering cards
     */
    pciaddress?: string;
    /**
     * Maker of the GPU, e.g. NVidia, AMD, Intel
     */
    vendor?: models_GPUVendor;
};

