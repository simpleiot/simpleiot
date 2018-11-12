port module Main exposing (Msg(..), main, update, view)

import Bootstrap.Accordion as Accordion
import Bootstrap.Card.Block as Block
import Bootstrap.Grid as Grid
import Bootstrap.Navbar as Navbar
import Browser
import Html exposing (Html, button, div, h1, text)
import Html.Attributes exposing (href)
import Html.Events exposing (onClick)
import Json.Decode as Decode
import Url.Builder as Url
import Time



main =
    Browser.document
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        }



-- Model

type alias Sample =
    { id : String
    , value: Float
    }

type alias Device =
    { id : String
    , description: String
    , ios : List Sample
    }


type alias Model =
    { navbarState : Navbar.State
    , accordionState : Accordion.State
    , devices : List Device
    }


type Msg
    = Increment
    | Decrement
    | NavbarMsg Navbar.State
    | AccordionMsg Accordion.State
    | Tick Time.Posix



-- Subscriptions


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Navbar.subscriptions model.navbarState NavbarMsg
        , Accordion.subscriptions model.accordionState AccordionMsg
        , Time.every 1000 Tick
        ]



-- The navbar needs to know the initial window size, so the inital state for a navbar requires a command to be run by the Elm runtime
-- Init


init : () -> ( Model, Cmd Msg )
init model =
    let
        ( navbarState, navbarCmd ) =
            Navbar.initialState NavbarMsg
    in
    ( { navbarState = navbarState
      , accordionState = Accordion.initialState
      , devices = []
      }
    , navbarCmd
    )



-- Update

urlDevices = Url.absolute ["v1", "devices"]

update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Increment ->
            ( model, Cmd.none )

        Decrement ->
            ( model, Cmd.none )

        NavbarMsg state ->
            ( { model | navbarState = state }, Cmd.none )

        AccordionMsg state ->
            ( { model | accordionState = state }, Cmd.none )

        Tick newTime ->
            ( model, Cmd.none )



-- View


view : Model -> Browser.Document Msg
view model =
    { title = "Simple • IoT"
    , body =
        [ div []
            [ menu model
            , mainContent model
            ]
        ]
    }


menu : Model -> Html Msg
menu model =
    Navbar.config NavbarMsg
        |> Navbar.withAnimation
        |> Navbar.brand [ href "#" ] [ text "Simple IoT" ]
        |> Navbar.items
            [ Navbar.itemLink [ href "#" ] [ text "Item 1" ]
            , Navbar.itemLink [ href "#" ] [ text "Item 2" ]
            ]
        |> Navbar.view model.navbarState


mainContent : Model -> Html Msg
mainContent model =
    Grid.container []
        [ h1 [] [ text "Devices" ]
        , devices model
        ]


devices : Model -> Html Msg
devices model =
    Accordion.config AccordionMsg
        |> Accordion.withAnimation
        |> Accordion.cards
            [ Accordion.card
                { id = "card1"
                , options = []
                , header =
                    Accordion.header [] <| Accordion.toggle [] [ text "Device #1" ]
                , blocks =
                    [ Accordion.block []
                        [ Block.text [] [ text "78°F" ] ]
                    ]
                }
            , Accordion.card
                { id = "card2"
                , options = []
                , header =
                    Accordion.header [] <| Accordion.toggle [] [ text "Device #2" ]
                , blocks =
                    [ Accordion.block []
                        [ Block.text [] [ text "75°F" ] ]
                    ]
                }
            ]
        |> Accordion.view model.accordionState
