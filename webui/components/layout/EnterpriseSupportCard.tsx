'use client'

import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { X } from 'lucide-react'

export function EnterpriseSupportCard() {
  const [hideSupportCard, setHideSupportCard] = useState(false)
  const [hasCheckedStorage, setHasCheckedStorage] = useState(false)

  useEffect(() => {
    const storedVisibility = localStorage.getItem('hideSupportCard')
    if (storedVisibility !== null) {
      setHideSupportCard(JSON.parse(storedVisibility))
    }
    setHasCheckedStorage(true)
  }, [])

  const handleClose = () => {
    setHideSupportCard(true)
    localStorage.setItem('hideSupportCard', 'true')
  }

  if (!hasCheckedStorage || hideSupportCard) return null

  return (
    <Card className="bg-sidebar-bg border-sidebar-border text-sidebar-text relative">
      <button
        onClick={handleClose}
        className="absolute top-2 right-2 text-gray-400 hover:text-white"
        aria-label="Close"
      >
        <X size={20} />
      </button>
      <CardHeader className="p-4">
        <CardTitle className="text-white">Get Support</CardTitle>
        <CardDescription className="text-gray-400">
          Get dedicated support for your organization's needs.
        </CardDescription>
      </CardHeader>
      <CardContent className="p-4 pt-0">
        <Button
          size="sm"
          className="w-full bg-button-primary text-white hover:bg-button-primaryHover"
          onClick={() => window.open('https://expanso.io/contact/', '_blank')}
        >
          Contact Expanso
        </Button>
      </CardContent>
    </Card>
  )
}
