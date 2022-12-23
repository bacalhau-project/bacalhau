import React, { FC, useRef, useCallback, useState, useMemo } from 'react'
import { SxProps } from '@mui/system'
import Box from '@mui/material/Box'
import ForceGraph2D from 'react-force-graph-2d'

import {
  ClusterMapResult,
} from '../../types'

import {
  getShortId,
} from '../../utils/job'

const NODE_R = 6

type CanvasCustomRenderMode = 'replace' | 'before' | 'after'

type NodeCallbackFunction = {
  (id: string): void
}

const ForceGraph: FC<{
  data: ClusterMapResult,
  size?: number,
  sx?: SxProps,
}> = ({
  data,
  size = 600,
  sx = {},
}) => {
  const fgRef = useRef()

  const [highlightNodes, setHighlightNodes] = useState(new Set())
  const [highlightLinks, setHighlightLinks] = useState(new Set())
  const [hoverNode, setHoverNode] = useState(null)
  const [clickedNode, setClickedNode] = useState(null)

  const useData = useMemo(() => {
    if(!data) return data    
    const useData = JSON.parse(JSON.stringify(data))
    const nodeMap = useData.nodes.reduce((acc: any, node: any) => {
      acc[node.id] = node
      return acc
    }, {})
    useData.links.forEach((link: any) => {
      const a = nodeMap[link.source];
      const b = nodeMap[link.target];
      !a.neighbors && (a.neighbors = []);
      !b.neighbors && (b.neighbors = []);
      a.neighbors.push(b);
      b.neighbors.push(a);

      !a.links && (a.links = []);
      !b.links && (b.links = []);
      a.links.push(link);
      b.links.push(link);
    })
    return useData
  }, [data])

  const updateHighlight = () => {
    setHighlightNodes(highlightNodes)
    setHighlightLinks(highlightLinks)
  }

  const handleNodeHover = (node: any) => {
    highlightNodes.clear()
    highlightLinks.clear()
    if (node) {
      highlightNodes.add(node)
      node.neighbors.forEach((neighbor: any) => highlightNodes.add(neighbor))
      node.links.forEach((link: any) => highlightLinks.add(link))
    }

    setHoverNode(node || null)
    updateHighlight()
  }

  const handleLinkHover = (link: any) => {
    highlightNodes.clear()
    highlightLinks.clear()

    if (link) {
      highlightLinks.add(link)
      highlightNodes.add(link.source)
      highlightNodes.add(link.target)
    }

    updateHighlight()
  }

  const paintRing = useCallback((node: any, ctx: any) => {
    // add ring just for highlighted nodes
    ctx.beginPath()
    ctx.arc(node.x, node.y, NODE_R * 1.4, 0, 2 * Math.PI, false)
    ctx.fillStyle = node === hoverNode ? 'red' : 'orange'
    ctx.fill()
  }, [hoverNode])

  return (
    <>
      <Box
        component="div"
        sx={{
          border: '1px solid #999',
          width: `${size}px`,
          height: `${size}px`,
          ...sx
        }}
      >
        <ForceGraph2D
          ref={fgRef}
          graphData={useData}
          width={size}
          height={size}
          nodeLabel={(node: any) => getShortId(node.id)}
          nodeRelSize={NODE_R}
          autoPauseRedraw={false}
          linkWidth={link => highlightLinks.has(link) ? 5 : 1}
          linkDirectionalParticles={4}
          linkDirectionalParticleWidth={(link: any) => highlightLinks.has(link) ? 4 : 0}
          nodeCanvasObjectMode={(node: any) => highlightNodes.has(node) ? 'before' : (undefined as unknown as CanvasCustomRenderMode)}
          nodeCanvasObject={paintRing}
          onNodeHover={handleNodeHover}
          onLinkHover={handleLinkHover}
        />
      </Box>
    </>
    
  )
}

export default ForceGraph
