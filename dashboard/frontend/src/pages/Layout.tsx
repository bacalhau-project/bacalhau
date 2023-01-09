import React, { FC, useState, useContext, useEffect, useMemo, useCallback } from 'react'
import bluebird from 'bluebird'
import { navigate } from 'hookrouter'
import { styled, useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
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
import IconButton from '@mui/material/IconButton'

import DvrIcon from '@mui/icons-material/Dvr'
import CategoryIcon from '@mui/icons-material/Category'
import AccountTreeIcon from '@mui/icons-material/AccountTree'
import MenuIcon from '@mui/icons-material/Menu'
import LoginIcon from '@mui/icons-material/Login'
import LogoutIcon from '@mui/icons-material/Logout'

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
  const [ mobileOpen, setMobileOpen ] = useState(false)

  const theme = useTheme()
  const bigScreen = useMediaQuery(theme.breakpoints.up('md'))

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen)
  };

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
    setMobileOpen(false)
    await user.logout()
    snackbar.success('Logout successful')
  }, [])

  const drawer = (
    <div>
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
            setMobileOpen(false)
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
            setMobileOpen(false)
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
            setMobileOpen(false)
          }}
        >
          <ListItemButton>
            <ListItemIcon>
              <CategoryIcon />
            </ListItemIcon>
            <ListItemText primary="Jobs" />
          </ListItemButton>
        </ListItem>
        <Divider />
        {
          user.user ? (
            <ListItem
              disablePadding
              onClick={ onLogout }
            >
              <ListItemButton>
                <ListItemIcon>
                  <LogoutIcon />
                </ListItemIcon>
                <ListItemText primary="Logout" />
              </ListItemButton>
            </ListItem>
          ) : (
            <ListItem
              disablePadding
              onClick={ () => {
                setLoginOpen(true) 
                setMobileOpen(false)
              }}
            >
              <ListItemButton>
                <ListItemIcon>
                  <LoginIcon />
                </ListItemIcon>
                <ListItemText primary="Login" />
              </ListItemButton>
            </ListItem>
          )
        }
        
      </List>
    </div>
  )

  const container = window !== undefined ? () => document.body : undefined

  useEffect(() => {
    user.initialise()
  }, [])

  return (
    <Box sx={{ display: 'flex' }} component="div">
      <CssBaseline />
      <AppBar
        elevation={ 1 }
        position="fixed"
        open
        color="default"
        sx={{
          width: { xs: '100%', sm: '100%', md: `calc(100% - ${drawerWidth}px)` },
          ml: { xs: '0px', sm: '0px', md: `${drawerWidth}px` },
        }}
      >
        <Toolbar
          sx={{
            pr: '24px', // keep right padding when drawer closed
            backgroundColor: '#fff'
          }}
        >
          {
            !bigScreen && (
              <>
                <IconButton
                  color="inherit"
                  aria-label="open drawer"
                  edge="start"
                  onClick={ handleDrawerToggle }
                  sx={{
                    mr: 1,
                    ml: 1,
                  }}
                >
                  <MenuIcon />
                </IconButton>
                <Logo
                  src="/img/logo.png"
                  sx={{
                    mr: 1,
                    ml: 1,
                  }}
                />
                <Typography
                  variant="h6"
                  sx={{
                    mr: 1,
                    ml: 1,
                  }}
                >
                  Bacalhau
                </Typography>
                <Typography
                  variant="h6"
                  sx={{
                    mr: 1,
                    ml: 1,
                  }}
                >
                  :
                </Typography>
              </>
              
            )
          }
          <Typography
            component="h1"
            variant="h6"
            color="inherit"
            noWrap
            sx={{
              flexGrow: 1,
              ml: 1,
              color: 'text.primary',
            }}
          >
            {route.title || 'Page'}
          </Typography>
          {
            bigScreen && (
              <>
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
              </>
            )
          }
        </Toolbar>
      </AppBar>
      <MuiDrawer
        container={container}
        variant="temporary"
        open={mobileOpen}
        onClose={handleDrawerToggle}
        ModalProps={{
          keepMounted: true, // Better open performance on mobile.
        }}
        sx={{
          display: { sm: 'block', md: 'none' },
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
        }}
      >
        {drawer}
      </MuiDrawer>
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: 'none', md: 'block' },
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
        }}
        open
      >
        {drawer}
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