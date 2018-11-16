port module Main exposing (Msg(..), main, update, view)

import Bootstrap.Accordion as Accordion
import Bootstrap.Alert as Alert
import Bootstrap.Button as Button
import Bootstrap.Card.Block as Block
import Bootstrap.Form as Form
import Bootstrap.Form.Checkbox as Checkbox
import Bootstrap.Form.Fieldset as Fieldset
import Bootstrap.Form.Input as Input
import Bootstrap.Form.Radio as Radio
import Bootstrap.Form.Select as Select
import Bootstrap.Form.Textarea as Textarea
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.ListGroup as ListGroup
import Bootstrap.Modal as Modal
import Bootstrap.Navbar as Navbar
import Browser
import Color exposing (Color)
import Html exposing (Html, button, div, h1, h4, span, text)
import Html.Attributes exposing (class, href, placeholder, style, type_)
import Html.Events exposing (onClick, onInput)
import Http
import Json.Decode as Decode
import List.Extra as ListExtra
import Material.Icons.Image exposing (edit)
import Round
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


type alias Device =
    { id : String
    , description : String
    , ios : List Sample
    }


type alias Model =
    { navbarState : Navbar.State
    , accordionState : Accordion.State
    , devices : List Device
    , editDeviceVisibility : Modal.Visibility
    , editDevice : Maybe Device
    }


type Msg
    = Increment
    | Decrement
    | NavbarMsg Navbar.State
    | AccordionMsg Accordion.State
    | Tick Time.Posix
    | UpdateDevices (Result Http.Error (List Device))
    | EditDevice String
    | EditDeviceClose
    | EditDeviceSave
    | EditDeviceChangeDescription String



-- Subscriptions


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Navbar.subscriptions model.navbarState NavbarMsg
        , Accordion.subscriptions model.accordionState AccordionMsg
        , Time.every 5000 Tick
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
      , editDeviceVisibility = Modal.hidden
      , editDevice = Nothing
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


deviceStateDecoder : Decode.Decoder Device
deviceStateDecoder =
    Decode.map3 Device
        (Decode.field "id" Decode.string)
        (Decode.field "description" Decode.string)
        (Decode.field "ios" samplesDecoder)


devicesDecoder : Decode.Decoder (List Device)
devicesDecoder =
    Decode.list deviceStateDecoder


getDevices : Cmd Msg
getDevices =
    Http.send UpdateDevices (Http.get urlDevices devicesDecoder)


findDevice : Model -> String -> Maybe Device
findDevice model id =
    ListExtra.find (\d -> d.id == id) model.devices


updateDevice : Model -> Maybe Device -> Model
updateDevice model device =
    case device of
        Nothing ->
            model

        Just deviceUpdate ->
            let
                index =
                    ListExtra.findIndex (\d -> d.id == deviceUpdate.id) model.devices

                devices =
                    case index of
                        Nothing ->
                            List.append model.devices [ deviceUpdate ]

                        Just i ->
                            ListExtra.setAt i deviceUpdate model.devices
            in
            { model | devices = devices }


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    -- uncomment the following to display model updates
    let
        --    _ =
        --        Debug.log "update: " msg
        _ =
            Debug.log "model: " model
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

        EditDevice id ->
            ( { model
                | editDeviceVisibility = Modal.shown
                , editDevice = findDevice model id
              }
            , Cmd.none
            )

        EditDeviceClose ->
            ( { model | editDeviceVisibility = Modal.hidden }, Cmd.none )

        EditDeviceSave ->
            ( updateDevice model model.editDevice, Cmd.none )

        EditDeviceChangeDescription desc ->
            case model.editDevice of
                Nothing ->
                    ( model, Cmd.none )

                Just device ->
                    let
                        oldEditDevice =
                            device

                        newEditDevice =
                            { oldEditDevice | description = desc }
                    in
                    ( { model | editDevice = Just newEditDevice }, Cmd.none )



-- View


view : Model -> Browser.Document Msg
view model =
    { title = "Simple • IoT"
    , body =
        [ div []
            [ menu model
            , mainContent model
            , renderEditDevice model
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
        , renderDevices model
        ]


renderDevices : Model -> Html Msg
renderDevices model =
    Accordion.config AccordionMsg
        |> Accordion.withAnimation
        |> Accordion.cards
            (List.map
                renderDevice
                model.devices
            )
        |> Accordion.view model.accordionState


renderDeviceSummary : Device -> String
renderDeviceSummary dev =
    dev.description ++ "(" ++ dev.id ++ ")"


renderDevice : Device -> Accordion.Card Msg
renderDevice dev =
    Accordion.card
        { id = dev.id
        , options = []
        , header =
            Accordion.header []
                (Accordion.toggle [] [ h4 [] [ text (renderDeviceSummary dev) ] ])
                |> Accordion.appendHeader [ button [ type_ "button", onClick (EditDevice dev.id), class "btn btn-light" ] [ edit Color.black 25 ] ]
        , blocks =
            [ renderIos dev.ios ]
        }


renderIos : List Sample -> Accordion.CardBlock Msg
renderIos samples =
    Accordion.listGroup
        (List.map
            (\s -> ListGroup.li [] [ text (s.id ++ ": " ++ Round.round 2 s.value) ])
            samples
        )


renderEditDevice : Model -> Html Msg
renderEditDevice model =
    case model.editDevice of
        Nothing ->
            Modal.config EditDeviceClose
                |> Modal.small
                |> Modal.h5 [] [ text "Warning!" ]
                |> Modal.body []
                    [ text "No device to edit" ]
                |> Modal.footer []
                    [ Button.button
                        [ Button.outlinePrimary
                        , Button.attrs [ onClick EditDeviceClose ]
                        ]
                        [ text "Cancel" ]
                    ]
                |> Modal.view model.editDeviceVisibility

        Just device ->
            Modal.config EditDeviceClose
                |> Modal.small
                |> Modal.h5 [] [ text ("Edit device (" ++ device.id ++ ")") ]
                |> Modal.body []
                    [ Form.form []
                        [ Form.group []
                            [ Form.label [] [ text "Device description" ]
                            , Input.text
                                [ Input.attrs
                                    [ placeholder "enter description"
                                    , onInput EditDeviceChangeDescription
                                    ]
                                ]
                            ]
                        ]
                    ]
                |> Modal.footer []
                    [ Button.button
                        [ Button.outlinePrimary
                        , Button.attrs [ onClick EditDeviceSave ]
                        ]
                        [ text "Save" ]
                    , Button.button
                        [ Button.outlineWarning
                        , Button.attrs [ onClick EditDeviceClose ]
                        ]
                        [ text "Cancel" ]
                    ]
                |> Modal.view model.editDeviceVisibility
