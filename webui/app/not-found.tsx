import React from 'react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '@/components/ui/card'
import { HomeIcon } from 'lucide-react'

export default function NotFound(): React.ReactElement {
  return (
    <div className="flex items-center justify-center h-full">
      <Card className="w-[350px] shadow-lg">
        <CardHeader className="text-center">
          <CardTitle className="text-3xl font-bold text-gray-800">
            404
          </CardTitle>
          <CardDescription className="text-xl text-gray-600">
            Page Not Found
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-center text-gray-600 mb-4">
            Oops! The page you&apos;re looking for doesn&apos;t exist.
          </p>
          <div className="flex justify-center">
            <span className="text-6xl">🕵️‍♂️</span>
          </div>
        </CardContent>
        <CardFooter className="flex justify-center">
          <Button asChild className="w-full">
            <Link href="/" className="flex items-center justify-center">
              <HomeIcon className="mr-2 h-4 w-4" /> Return Home
            </Link>
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
