module Pages.Devices exposing (Model, Msg, page)

import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Html.Events
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Json.Encode as Encode
import Sample exposing (Sample, encodeSample, renderSample, sampleDecoder)
import Spa.Page
import Spa.Types as Types
import Time
import Url.Builder as Url
import Utils.Spa exposing (Page)
import Utils.Styles exposing (size, palette)


page : Page Params.Devices Model Msg model msg appMsg
page =
    Spa.Page.element
        { title = always "Devices"
        , init = always init
        , update = update
        , subscriptions = subscriptions
        , view = always view
        }



-- INIT


type alias Model =
    { devices : List Device
    , deviceEdits : Dict String String
    }


init : Params.Devices -> ( Model, Cmd Msg )
init _ =
    ( { devices = []
      , deviceEdits = Dict.empty
      }
    , Cmd.none
    )



-- UPDATE


type Msg
    = Tick Time.Posix
    | UpdateDevices (Result Http.Error (List Device))
    | EditDeviceDescription DeviceEdit
    | PostDeviceConfig String DeviceConfig
    | DiscardEditedDeviceDescription String
    | DeviceConfigPosted String (Result Http.Error Response)


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg )
update context msg model =
    case msg of
        Tick _ ->
            ( model
            , case context.global of
                Global.SignedIn sess ->
                    getDevices sess.authToken

                Global.SignedOut _ ->
                    Cmd.none
            )

        UpdateDevices (Ok devices) ->
            ( { model | devices = devices }
            , Cmd.none
            )

        EditDeviceDescription { id, description } ->
            ( { model | deviceEdits = Dict.insert id description model.deviceEdits }
            , Cmd.none
            )

        PostDeviceConfig id config ->
            ( model
            , case context.global of
                Global.SignedIn sess ->
                    postDeviceConfig sess.authToken id config

                Global.SignedOut _ ->
                    Cmd.none
            )

        DeviceConfigPosted id (Ok _) ->
            ( { model | deviceEdits = Dict.remove id model.deviceEdits }
            , Cmd.none
            )

        DiscardEditedDeviceDescription id ->
            ( { model | deviceEdits = Dict.remove id model.deviceEdits }
            , Cmd.none
            )

        _ ->
            ( model
            , Cmd.none
            )


urlDevices =
    Url.absolute [ "v1", "devices" ] []


type alias Device =
    { id : String
    , config : DeviceConfig
    , state : DeviceState
    }


type alias DeviceEdit =
    { id : String
    , description : String
    }


type alias DeviceConfig =
    { description : String
    }


type alias DeviceState =
    { ios : List Sample
    }


devicesDecoder : Decode.Decoder (List Device)
devicesDecoder =
    Decode.list deviceDecoder


deviceDecoder : Decode.Decoder Device
deviceDecoder =
    Decode.map3 Device
        (Decode.field "id" Decode.string)
        (Decode.field "config" deviceConfigDecoder)
        (Decode.field "state" deviceStateDecoder)


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


getDevices : String -> Cmd Msg
getDevices token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = urlDevices
        , expect = Http.expectJson UpdateDevices devicesDecoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }



-- SUBSCRIPTIONS


subscriptions : Types.PageContext route Global.Model -> Model -> Sub Msg
subscriptions context model =
    -- TODO: Subscribe to ticker only when context.global == Global.SignedIn
    Sub.batch
        [ Time.every 1000 Tick
        ]



-- VIEW


view : Model -> Element Msg
view model =
    column
        [ width fill, spacing 32 ]
        [ el [ padding 16, Font.size 24 ] <| text "Devices"
        , viewDevices model
        ]


viewDevices : Model -> Element Msg
viewDevices model =
    column
        [ width fill
        , spacing 24
        ]
    <|
        List.map (viewDevice model.deviceEdits) model.devices


viewDevice : Dict String String -> Device -> Element Msg
viewDevice edits device =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color palette.black
        , spacing 6
        ]
        [ viewDeviceId device.id
        , viewDeviceDescription edits device
        , viewIoList device.state.ios
        ]


viewDeviceDescription : Dict String String -> Device -> Element Msg
viewDeviceDescription edits device =
    descriptionField
        device.id
        { description = deviceDescription edits device }
        (modified edits device)


viewDeviceId : String -> Element Msg
viewDeviceId id =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text id


viewIoList : List Sample -> Element Msg
viewIoList ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (renderSample >> text) ios


deviceDescription : Dict String String -> Device -> String
deviceDescription edits device =
    case Dict.get device.id edits of
        Just desc ->
            desc

        Nothing ->
            device.config.description


modified : Dict String String -> Device -> Bool
modified edits device =
    case Dict.get device.id edits of
        Just desc ->
            desc /= device.config.description

        Nothing ->
            False


descriptionField : String -> DeviceConfig -> Bool -> Element Msg
descriptionField id config modded =
    Input.text
        (fieldAttrs
            modded
            (PostDeviceConfig id config)
            (DiscardEditedDeviceDescription id)
        )
        { onChange =
            \d ->
                EditDeviceDescription
                    { id = id
                    , description = d
                    }
        , text = config.description
        , placeholder =
            Just <|
                Input.placeholder
                    [ Font.italic
                    , Font.color palette.gray
                    ]
                <|
                    text "description"
        , label = Input.labelHidden "Description"
        }


fieldAttrs : Bool -> Msg -> Msg -> List (Attribute Msg)
fieldAttrs modded save discard =
    [ padding 16
    , width fill
    , Border.width 0
    , Border.rounded 0
    , focused [ Background.color palette.yellow ]
    ]
        ++ (if modded then
                [ Background.color palette.orange
                , onEnter save
                , below <|
                    buttonRow
                        [ button "discard" palette.pale discard
                        , button "save" palette.green save
                        ]
                ]

            else
                [ Background.color palette.pale ]
           )


buttonRow : List (Element Msg) -> Element Msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button : String -> Color -> Msg -> Element Msg
button label color action =
    Input.button
        [ Background.color color
        , padding 16
        , width fill
        , Border.rounded 32
        ]
        { onPress = Just action
        , label = el [ centerX ] <| text label
        }


onEnter : msg -> Attribute msg
onEnter msg =
    htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Enter" then
                            Decode.succeed msg

                        else
                            Decode.fail "Not the enter key"
                    )
            )
       )


deviceConfigEncoder : DeviceConfig -> Encode.Value
deviceConfigEncoder deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description ) ]


postDeviceConfig : String -> String -> DeviceConfig -> Cmd Msg
postDeviceConfig token id config =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id, "config" ] []
        , expect = Http.expectJson (DeviceConfigPosted id) responseDecoder
        , body = config |> deviceConfigEncoder |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""
