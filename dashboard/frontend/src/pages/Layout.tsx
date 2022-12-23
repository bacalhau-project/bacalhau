import React, { FC, useState, useContext, useEffect, useMemo, useCallback } from 'react'
import bluebird from 'bluebird'
import { navigate } from 'hookrouter'
import { styled } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import MuiDrawer from '@mui/material/Drawer'
import Grid from '@mui/material/Grid'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import MuiAppBar, { AppBarProps as MuiAppBarProps } from '@mui/material/AppBar'
import Toolbar from '@mui/material/Toolbar'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import Divider from '@mui/material/Divider'
import Container from '@mui/material/Container'
import Stack from '@mui/material/Stack'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import Link from '@mui/material/Link'

import DvrIcon from '@mui/icons-material/Dvr'
import CategoryIcon from '@mui/icons-material/Category'
import AccountTreeIcon from '@mui/icons-material/AccountTree'

import { RouterContext } from '../contexts/router'
import { UserContext } from '../contexts/user'
import Snackbar from '../components/system/Snackbar'
import Window from '../components/widgets/Window'
import GlobalLoading from '../components/system/GlobalLoading'
import useSnackbar from '../hooks/useSnackbar'
import useLoadingErrorHandler from '../hooks/useLoadingErrorHandler'

function Copyright(props: any) {
  return (
    <Typography variant="body2" color="text.secondary" align="center" {...props}>
      {'Copyright © '}
      <Link color="inherit" href="https://www.bacalhau.org/">
        Bacalhau
      </Link>{' '}
      {new Date().getFullYear()}
      {'.'}
    </Typography>
  )
}

const drawerWidth: number = 240

interface AppBarProps extends MuiAppBarProps {
  open?: boolean
}

const Logo = styled('img')({
  height: '50px',
})

const AppBar = styled(MuiAppBar, {
  shouldForwardProp: (prop) => prop !== 'open',
})<AppBarProps>(({ theme, open }) => ({
  zIndex: theme.zIndex.drawer + 1,
  transition: theme.transitions.create(['width', 'margin'], {
    easing: theme.transitions.easing.sharp,
    duration: theme.transitions.duration.leavingScreen,
  }),
  ...(open && {
    marginLeft: drawerWidth,
    width: `calc(100% - ${drawerWidth}px)`,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
  }),
}))

const Drawer = styled(MuiDrawer, { shouldForwardProp: (prop) => prop !== 'open' })(
  ({ theme, open }) => ({
    '& .MuiDrawer-paper': {
      position: 'relative',
      whiteSpace: 'nowrap',
      width: drawerWidth,
      transition: theme.transitions.create('width', {
        easing: theme.transitions.easing.sharp,
        duration: theme.transitions.duration.enteringScreen,
      }),
      boxSizing: 'border-box',
      ...(!open && {
        overflowX: 'hidden',
        transition: theme.transitions.create('width', {
          easing: theme.transitions.easing.sharp,
          duration: theme.transitions.duration.leavingScreen,
        }),
        width: theme.spacing(7),
        [theme.breakpoints.up('sm')]: {
          width: theme.spacing(9),
        },
      }),
    },
  }),
)

const Layout: FC = () => {
  const route = useContext(RouterContext)
  const user = useContext(UserContext)
  const snackbar = useSnackbar()
  const [ username, setUsername ] = useState('')
  const [ password, setPassword ] = useState('')
  const [ loginOpen, setLoginOpen ] = useState(false)

  const onLogin = useCallback(async () => {
    const result = await user.login(username, password)
    if (result) {
      snackbar.success('Login successful')
      setLoginOpen(false)
    } else {
      snackbar.error('Incorrect details')
    }
  }, [
    username,
    password,
  ])

  const onLogout = useCallback(async () => {
    await user.logout()
    snackbar.success('Logout successful')
  }, [])

  useEffect(() => {
    user.initialise()
  }, [])

  return (
    <Box sx={{ display: 'flex' }} component="div">
      <CssBaseline />
      <AppBar elevation={ 1 } position="absolute" open color="default">
        <Toolbar
          sx={{
            pr: '24px', // keep right padding when drawer closed
            backgroundColor: '#fff'
          }}
        >
          <Typography
            component="h1"
            variant="h6"
            color="inherit"
            noWrap
            sx={{
              flexGrow: 1,
              marginLeft: '16px',
              color: 'text.primary',
            }}
          >
            {route.title || 'Page'}
          </Typography>
          {
            user.user ? (
              <Stack
                direction="row"
                spacing={2}
                justifyContent="center"
                alignItems="center"
              >
                <Typography variant="body1">
                  {
                    user.user.username
                  }
                </Typography>
                <Button
                  color="primary"
                  variant="outlined"
                  onClick={ onLogout }
                >
                  Logout
                </Button>
              </Stack>
              
            ) : (
              <Button
                color="primary"
                variant="outlined"
                onClick={ () => setLoginOpen(true) }
              >
                Login
              </Button>
            )
          }
          
        </Toolbar>
      </AppBar>
      <Drawer variant="permanent" open>
        <Toolbar
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-start',
            px: [1],
          }}
        >
          <Logo
            src="/img/logo.png"
          />
          <Typography variant="h6">
            Bacalhau
          </Typography>
        </Toolbar>
        <Divider />
        <List>
          <ListItem
            disablePadding
            selected={route.id === 'home'}
            onClick={ () => {
              navigate('/')
            }}
          >
            <ListItemButton>
              <ListItemIcon>
                <DvrIcon />
              </ListItemIcon>
              <ListItemText primary="Dashboard" />
            </ListItemButton>
          </ListItem>
          <ListItem
            disablePadding
            selected={route.id === 'network'}
            onClick={ () => {
              navigate('/network')
            }}
          >
            <ListItemButton>
              <ListItemIcon>
                <AccountTreeIcon />
              </ListItemIcon>
              <ListItemText primary="Network" />
            </ListItemButton>
          </ListItem>
          <ListItem
            disablePadding
            selected={route.id.indexOf('jobs') === 0}
            onClick={ () => {
              navigate('/jobs')
            }}
          >
            <ListItemButton>
              <ListItemIcon>
                <CategoryIcon />
              </ListItemIcon>
              <ListItemText primary="Jobs" />
            </ListItemButton>
          </ListItem>
        </List>
      </Drawer>
      <Box
        component="main"
        sx={{
          backgroundColor: (theme) =>
            theme.palette.mode === 'light'
              ? theme.palette.grey[100]
              : theme.palette.grey[900],
          flexGrow: 1,
          height: '100vh',
          overflow: 'auto',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <Box
          component="div"
          sx={{
            flexGrow: 0,
          }}
        >
          <Toolbar />
        </Box>
        <Box
          component="div"
          sx={{
            flexGrow: 1,
          }}
        >
          {route.render()}
        </Box>
        <Box
          component="div"
          sx={{
            flexGrow: 0,
          }}
        >
          <Container maxWidth={'xl'} sx={{ mt: 4, mb: 4 }}>
            <Copyright sx={{ pt: 4 }} />
          </Container>
        </Box>
      </Box>
      {
        loginOpen && (
          <Window
            open
            size="md"
            title="Login"
            submitTitle="Login"
            withCancel
            onCancel={ () => setLoginOpen(false) }
            onSubmit={ onLogin }
          >
            <Box
              sx={{
                p: 2,
              }}
            >
              <Grid container spacing={ 0 }>
                <Grid item xs={ 12 }>
                  <TextField
                    fullWidth
                    label="Username"
                    name="username"
                    required
                    size="small"
                    variant="outlined"
                    value={ username }
                    onChange={(e) => setUsername(e.target.value)}
                  />
                </Grid>
                <Grid item xs={ 12 }>
                  <TextField
                    fullWidth
                    type="password"
                    label="Password"
                    name="password"
                    required
                    size="small"
                    variant="outlined"
                    sx={{
                      mt: 2,
                    }}
                    value={ password }
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </Grid>
              </Grid>
            </Box>
          </Window>
        )
      }
      <Snackbar />
      <GlobalLoading />
    </Box>
  )
}

export default Layout