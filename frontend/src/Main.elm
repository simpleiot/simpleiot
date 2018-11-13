port module Main exposing (Msg(..), main, update, view)

import Bootstrap.Accordion as Accordion
import Bootstrap.Card.Block as Block
import Bootstrap.Grid as Grid
import Bootstrap.ListGroup as ListGroup
import Bootstrap.Navbar as Navbar
import Browser
import Html exposing (Html, button, div, h1, text)
import Html.Attributes exposing (href)
import Html.Events exposing (onClick)
import Http
import Json.Decode as Decode
import Time
import Url.Builder as Url


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
    , value : Float
    , time : String
    }


type alias DeviceState =
    { id : String
    , description : String
    , ios : List Sample
    }


type alias Model =
    { navbarState : Navbar.State
    , accordionState : Accordion.State
    , devices : List DeviceState
    }


type Msg
    = Increment
    | Decrement
    | NavbarMsg Navbar.State
    | AccordionMsg Accordion.State
    | Tick Time.Posix
    | UpdateDevices (Result Http.Error (List DeviceState))



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


urlDevices =
    Url.absolute [ "v1", "devices" ] []


sampleDecoder : Decode.Decoder Sample
sampleDecoder =
    Decode.map3 Sample
        (Decode.field "id" Decode.string)
        (Decode.field "value" Decode.float)
        (Decode.field "time" Decode.string)


samplesDecoder : Decode.Decoder (List Sample)
samplesDecoder =
    Decode.list sampleDecoder


deviceStateDecoder : Decode.Decoder DeviceState
deviceStateDecoder =
    Decode.map3 DeviceState
        (Decode.field "id" Decode.string)
        (Decode.field "description" Decode.string)
        (Decode.field "ios" samplesDecoder)


devicesDecoder : Decode.Decoder (List DeviceState)
devicesDecoder =
    Decode.list deviceStateDecoder


getDevices =
    Http.send UpdateDevices (Http.get urlDevices devicesDecoder)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    let
        _ =
            Debug.log "update: " msg
    in
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
            ( model, getDevices )

        UpdateDevices result ->
            case result of
                Ok devicesUpdate ->
                    ( { model | devices = devicesUpdate }, Cmd.none )

                Err _ ->
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


ios : List Sample -> Accordion.CardBlock Msg
ios samples =
    Accordion.listGroup
        (List.map
            (\s -> ListGroup.li [] [ text (s.id ++ ": " ++ String.fromFloat s.value) ])
            samples
        )


device : DeviceState -> Accordion.Card Msg
device dev =
    Accordion.card
        { id = dev.id
        , options = []
        , header =
            Accordion.header [] <| Accordion.toggle [] [ text dev.id ]
        , blocks =
            [ ios dev.ios ]
        }


devices : Model -> Html Msg
devices model =
    Accordion.config AccordionMsg
        |> Accordion.withAnimation
        |> Accordion.cards
            (List.map
                device
                model.devices
            )
        |> Accordion.view model.accordionState
