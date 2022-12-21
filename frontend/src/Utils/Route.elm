module Utils.Route exposing
    ( fromUrl
    , navigate
    )

import Browser.Navigation as Nav
import Gen.Route as Route exposing (Route)
import Url exposing (Url)


navigate : Nav.Key -> Route -> Cmd msg
navigate key route =
    Nav.pushUrl key (Route.toHref route)


fromUrl : Url -> Route
fromUrl =
    Route.fromUrl
