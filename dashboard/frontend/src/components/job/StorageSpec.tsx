import { FC } from "react";
import { StorageSpec } from "../../types";
import { Stack, Typography } from "@mui/material";

const StorageSpecRow: FC<{
    spec: StorageSpec,
    name?: string,
}> = ({
    spec,
    name,
}) => {
    const href = spec.URL || `https://ipfs.io/ipfs/${spec.CID}`
    const size = spec.Metadata && spec.Metadata["size"]
    const count = spec.Metadata && spec.Metadata["count"]
    return <Stack direction="column" key={name || spec.Name} sx={{overflowWrap: "anywhere"}}>
        <Typography variant="caption">
        <a target="_blank" href={href}>
            {name || spec.Name || spec.URL || spec.CID}
        </a>
        </Typography>
        {(size && count) && (
            <Typography variant="caption" color="#333">
                {count} files, {size} bytes
            </Typography>
        )}
    </Stack>
}

export default StorageSpecRow
