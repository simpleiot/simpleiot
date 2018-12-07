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
import Html exposing (Html, button, div, h1, h4, img, span, text)
import Html.Attributes exposing (class, height, href, placeholder, src, style, type_, value, width)
import Html.Events exposing (onClick, onInput)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Json.Encode as Encode
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


type alias Response =
    { success : Bool
    , error : String
    }


type alias Sample =
    { id : String
    , value : Float
    , time : String
    }


type alias Device =
    { id : String
    , config : DeviceConfig
    , state : DeviceState
    }


type alias DeviceConfig =
    { description : String
    }


type alias DeviceState =
    { ios : List Sample
    }


type alias Devices =
    { devices : List Device
    , dirty : Bool
    }


type alias DeviceEdits =
    { device : Maybe Device
    , visibility : Modal.Visibility
    }


type alias Model =
    { navbarState : Navbar.State
    , accordionState : Accordion.State
    , devices : Devices
    , deviceEdits : DeviceEdits
    }


type Msg
    = Increment
    | Decrement
    | NavbarMsg Navbar.State
    | AccordionMsg Accordion.State
    | Tick Time.Posix
    | UpdateDevices (Result Http.Error (List Device))
    | DeviceConfigPosted (Result Http.Error Response)
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
      , devices = { devices = [], dirty = False }
      , deviceEdits = { device = Nothing, visibility = Modal.hidden }
      }
    , navbarCmd
    )



-- Update


urlDevices =
    Url.absolute [ "v1", "devices" ] []


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""


sampleDecoder : Decode.Decoder Sample
sampleDecoder =
    Decode.map3 Sample
        (Decode.field "id" Decode.string)
        (Decode.field "value" Decode.float)
        (Decode.field "time" Decode.string)


samplesDecoder : Decode.Decoder (List Sample)
samplesDecoder =
    Decode.list sampleDecoder


deviceConfigDecoder : Decode.Decoder DeviceConfig
deviceConfigDecoder =
    Decode.map DeviceConfig
        (Decode.field "description" Decode.string)


deviceStateDecoder : Decode.Decoder DeviceState
deviceStateDecoder =
    Decode.map DeviceState
        (Decode.field "ios" samplesDecoder)


deviceDecoder : Decode.Decoder Device
deviceDecoder =
    Decode.map3 Device
        (Decode.field "id" Decode.string)
        (Decode.field "config" deviceConfigDecoder)
        (Decode.field "state" deviceStateDecoder)


devicesDecoder : Decode.Decoder (List Device)
devicesDecoder =
    Decode.list deviceDecoder


apiGetDevices : Cmd Msg
apiGetDevices =
    Http.send UpdateDevices (Http.get urlDevices devicesDecoder)


deviceConfigEncoder : DeviceConfig -> Encode.Value
deviceConfigEncoder deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description )
        ]


apiPostDeviceConfig : String -> DeviceConfig -> Cmd Msg
apiPostDeviceConfig id config =
    let
        body =
            config |> deviceConfigEncoder |> Http.jsonBody

        url =
            Url.absolute [ "v1", "devices", id, "config" ] []
    in
    Http.send DeviceConfigPosted (Http.post url body responseDecoder)


findDevice : List Device -> String -> Maybe Device
findDevice devices id =
    ListExtra.find (\d -> d.id == id) devices


updateDevice : List Device -> Maybe Device -> List Device
updateDevice devices device =
    case device of
        Nothing ->
            devices

        Just deviceUpdate ->
            let
                index =
                    ListExtra.findIndex (\d -> d.id == deviceUpdate.id) devices

                devicesModified =
                    case index of
                        Nothing ->
                            List.append devices [ deviceUpdate ]

                        Just i ->
                            ListExtra.setAt i deviceUpdate devices
            in
            devicesModified


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    -- uncomment the following to display model updates
    -- let
    --    _ =
    --        Debug.log "update: " msg
    -- _ =
    --    Debug.log "model: " model
    -- in
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
            ( model, apiGetDevices )

        UpdateDevices result ->
            case model.devices.dirty of
                True ->
                    ( model, Cmd.none )

                False ->
                    case result of
                        Ok devicesUpdate ->
                            ( { model | devices = { devices = devicesUpdate, dirty = False } }, Cmd.none )

                        Err err ->
                            let
                                _ =
                                    Debug.log "UpdateDevices error: " err
                            in
                            ( model, Cmd.none )

        DeviceConfigPosted result ->
            let
                devices =
                    model.devices

                newDevices =
                    { devices | dirty = False }

                newModel =
                    { model | devices = newDevices }
            in
            case Debug.log "DeviceConfigPosted" result of
                Ok string ->
                    ( newModel, Cmd.none )

                Err err ->
                    let
                        _ =
                            Debug.log "DeviceConfigPosted error: " err
                    in
                    ( newModel, Cmd.none )

        EditDevice id ->
            ( { model
                | deviceEdits = { visibility = Modal.shown, device = findDevice model.devices.devices id }
              }
            , Cmd.none
            )

        EditDeviceClose ->
            ( { model
                | deviceEdits =
                    { visibility = Modal.hidden
                    , device = Nothing
                    }
              }
            , Cmd.none
            )

        EditDeviceSave ->
            ( { model
                | devices =
                    { devices = updateDevice model.devices.devices model.deviceEdits.device
                    , dirty = True
                    }
                , deviceEdits = { device = model.deviceEdits.device, visibility = Modal.hidden }
              }
            , case model.deviceEdits.device of
                Nothing ->
                    Cmd.none

                Just dev ->
                    apiPostDeviceConfig dev.id dev.config
            )

        EditDeviceChangeDescription desc ->
            case model.deviceEdits.device of
                Nothing ->
                    ( model, Cmd.none )

                Just device ->
                    let
                        deviceConfig =
                            device.config

                        newDeviceConfig =
                            { deviceConfig | description = desc }

                        newDevice =
                            { device | config = newDeviceConfig }

                        deviceEdits =
                            model.deviceEdits

                        newDeviceEdits =
                            { deviceEdits | device = Just newDevice }
                    in
                    ( { model | deviceEdits = newDeviceEdits }, Cmd.none )



-- View


view : Model -> Browser.Document Msg
view model =
    { title = "Simple • IoT"
    , body =
        [ div []
            [ menu model
            , mainContent model
            , renderEditDevice model.deviceEdits
            ]
        ]
    }


menu : Model -> Html Msg
menu model =
    Navbar.config NavbarMsg
        |> Navbar.withAnimation
        |> Navbar.brand [ href "#" ] [ img [ src "/public/simple-iot-logo.png", width 50, height 50 ] [] ]
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
                model.devices.devices
            )
        |> Accordion.view model.accordionState


renderDeviceSummary : Device -> String
renderDeviceSummary dev =
    dev.config.description ++ " (" ++ dev.id ++ ")"


renderDevice : Device -> Accordion.Card Msg
renderDevice dev =
    Accordion.card
        { id = dev.id
        , options = []
        , header =
            Accordion.header []
                (Accordion.toggle [] [ h4 [] [ text (renderDeviceSummary dev) ] ])
                |> Accordion.appendHeader
                    [ button
                        [ type_ "button"
                        , onClick (EditDevice dev.id)
                        , class "btn btn-light"
                        ]
                        [ edit Color.black 25 ]
                    ]
        , blocks = [ renderIos dev.state.ios ]
        }


renderIos : List Sample -> Accordion.CardBlock Msg
renderIos samples =
    Accordion.listGroup
        (List.map
            (\s -> ListGroup.li [] [ text (s.id ++ ": " ++ Round.round 2 s.value) ])
            samples
        )


renderEditDevice : DeviceEdits -> Html Msg
renderEditDevice deviceEdits =
    case deviceEdits.device of
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
                |> Modal.view deviceEdits.visibility

        Just device ->
            Modal.config EditDeviceClose
                |> Modal.small
                |> Modal.h5 [] [ text ("Edit device (" ++ device.id ++ ")") ]
                |> Modal.body []
                    [ Form.group []
                        [ Form.label [] [ text "Device description" ]
                        , Input.text
                            [ Input.attrs
                                [ placeholder "enter description"
                                , onInput EditDeviceChangeDescription
                                , value device.config.description
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
                |> Modal.view deviceEdits.visibility
